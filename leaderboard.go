package main

import (
	"encoding/json"
	"os"
	"sort"
	"time"
)

type LeaderboardEntry struct {
	Name       string        `json:"name"`
	Time       time.Duration `json:"time"`
	Difficulty Difficulty    `json:"difficulty"`
	Date       time.Time     `json:"date"`
}

type Leaderboard struct {
	Entries []LeaderboardEntry
}

func NewLeaderboard() *Leaderboard {
	return &Leaderboard{
		Entries: []LeaderboardEntry{},
	}
}

func (l *Leaderboard) AddEntry(name string, duration time.Duration, difficulty Difficulty) {
	entry := LeaderboardEntry{
		Name:       name,
		Time:       duration,
		Difficulty: difficulty,
		Date:       time.Now(),
	}
	l.Entries = append(l.Entries, entry)
	// Sort entries if needed
	l.SaveToFile("sudoku_leaderboard.json")
	sort.Slice(l.Entries, func(i, j int) bool {
		return l.Entries[i].Time < l.Entries[j].Time
	})
}

func (l *Leaderboard) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (l *Leaderboard) DeleteEntry(index int) {
	if index >= 0 && index < len(l.Entries) {
		// Remove the entry at the specified index
		l.Entries = append(l.Entries[:index], l.Entries[index+1:]...)
	}
	sort.Slice(l.Entries, func(i, j int) bool {
		return l.Entries[i].Time < l.Entries[j].Time
	})
}

func LoadLeaderboardFromFile(filename string) (*Leaderboard, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return NewLeaderboard(), nil
		}
		return nil, err
	}

	var leaderboard Leaderboard
	err = json.Unmarshal(data, &leaderboard)
	if err != nil {
		return nil, err
	}
	return &leaderboard, nil
}

func (l *Leaderboard) GetTopScores(difficulty Difficulty, limit int) []LeaderboardEntry {
	var filteredEntries []LeaderboardEntry
	for _, entry := range l.Entries {
		if entry.Difficulty == difficulty {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	sort.Slice(filteredEntries, func(i, j int) bool {
		return filteredEntries[i].Time < filteredEntries[j].Time
	})

	if len(filteredEntries) > limit {
		filteredEntries = filteredEntries[:limit]
	}

	return filteredEntries
}
