// Package ratelimit はインメモリのスライディングウィンドウ式レートリミッタを提供する。
//
// admin 単位で「過去 1 時間」「過去 24 時間」の Claude 呼び出し回数を上限と比較する。
// 制限値はプランごとに異なるため、プラン側で決まる Limits を都度受け取る設計。
package ratelimit

import (
	"sync"
	"time"
)

// Limits はプランごとのレート制限値。
type Limits struct {
	Hourly int // 1 時間あたりの上限
	Daily  int // 24 時間あたりの上限
}

// Limiter は adminID をキーに過去のタイムスタンプを保持し、
// Allow 呼び出しごとにスライディングウィンドウで判定する。
//
// 単一プロセスのインメモリ運用のみ。複数プロセスで共有したい場合は
// Redis などに置き換える前提（現状 Fly.io single instance なので不要）。
type Limiter struct {
	mu      sync.Mutex
	history map[string][]time.Time // adminID → 過去のタイムスタンプ（昇順、24h より古いものは捨てる）
}

// New は空の Limiter を返す。
func New() *Limiter {
	return &Limiter{
		history: make(map[string][]time.Time),
	}
}

// Allow は adminID + Limits + now を受け取り、許可するか判定する。
//
// 動作:
//  1. 24 時間より古いタイムスタンプを破棄する
//  2. 1 時間以内 / 24 時間以内それぞれカウント
//  3. どちらかの上限に達していたら false（タイムスタンプは追加しない）
//  4. 許可なら now を追加して true
//
// テスト時は now を直接指定することで時計の進行を制御できる。
func (l *Limiter) Allow(adminID string, limits Limits, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	hourAgo := now.Add(-time.Hour)
	dayAgo := now.Add(-24 * time.Hour)

	// 1. 24h より古いタイムスタンプを捨てる
	times := dropOlderThan(l.history[adminID], dayAgo)

	// 2. カウント（1 ループで両方数える）
	var inHour int
	for _, t := range times {
		if t.After(hourAgo) {
			inHour++
		}
	}
	inDay := len(times)

	// 3. 上限到達なら拒否（時刻は追加しない）。ただし古いものを捨てた状態は書き戻す
	if inHour >= limits.Hourly || inDay >= limits.Daily {
		l.saveOrDelete(adminID, times)
		return false
	}

	// 4. 許可：now を追加
	l.saveOrDelete(adminID, append(times, now))
	return true
}

// dropOlderThan は cutoff より前 (cutoff 以前) のタイムスタンプを除外して返す。
// times は昇順前提。先頭から二分探索でも良いが、件数が小さい (≦daily) ので線形で十分。
func dropOlderThan(times []time.Time, cutoff time.Time) []time.Time {
	for i, t := range times {
		if t.After(cutoff) {
			// times[:i] を捨てて times[i:] を返す。
			// 既存スライスを書き換えると並行アクセス時に問題になりうるので
			// 新しいスライスにコピーする。
			out := make([]time.Time, len(times)-i)
			copy(out, times[i:])
			return out
		}
	}
	// 全て cutoff 以前なら空にする
	return nil
}

// saveOrDelete はタイムスタンプ列を保存する。空ならメモリ節約のため map から削除。
func (l *Limiter) saveOrDelete(adminID string, times []time.Time) {
	if len(times) == 0 {
		delete(l.history, adminID)
		return
	}
	l.history[adminID] = times
}
