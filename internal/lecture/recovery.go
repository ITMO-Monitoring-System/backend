package lecture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"monitoring_backend/internal/domain"
)

const recoveryInterval = 5 * time.Second
const runStartGrace = 30 * time.Second

type recognizingLecture struct {
	LectureID int64  `json:"lecture_id"`
	Running   *bool  `json:"running"`
	InQueue   string `json:"in_queue"`
	OutQueue  string `json:"out_queue"`
}

// Maintain recovers result consumers after a backend restart and closes runs
// whose stop notification was lost. Unreachable/invalid status never means stopped.
func (m *Manager) Maintain(ctx context.Context, statusURL string) {
	defer m.closeConsumers()
	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(recoveryInterval)
	defer ticker.Stop()
	for {
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := m.reconcile(checkCtx, client, statusURL, time.Now())
		cancel()
		if err != nil && ctx.Err() == nil {
			slog.Warn("lecture_recovery_failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (m *Manager) reconcile(ctx context.Context, client *http.Client, statusURL string, checkedAt time.Time) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("recognizing status: HTTP %d", resp.StatusCode)
	}
	var snapshot struct {
		OK       bool                  `json:"ok"`
		Lectures *[]recognizingLecture `json:"lectures"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&snapshot); err != nil {
		return fmt.Errorf("recognizing status: %w", err)
	}
	if !snapshot.OK || snapshot.Lectures == nil {
		return errors.New("incomplete recognizing status")
	}
	for _, item := range *snapshot.Lectures {
		if item.LectureID <= 0 || item.Running == nil || (*item.Running && (item.InQueue == "" || item.OutQueue == "")) {
			return errors.New("invalid recognizing lecture status")
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	live := make(map[int64]map[string]string)
	for _, item := range *snapshot.Lectures {
		if !*item.Running {
			continue
		}
		// Tracking appends the UUID of the run to its input queue name.
		// Legacy queues have no UUID suffix and use the empty run ID.
		runID := inputQueueRunID(item.InQueue)
		if err := m.lectures.SetRunFinished(ctx, item.LectureID, runID, false); err != nil {
			if errors.Is(err, domain.ErrLectureRunFinished) || errors.Is(err, pgx.ErrNoRows) {
				continue // A stop may have committed while the HTTP snapshot was in flight.
			}
			return err
		}
		if live[item.LectureID] == nil {
			live[item.LectureID] = make(map[string]string)
		}
		live[item.LectureID][runID] = item.OutQueue
	}
	runs, err := m.lectures.ListUnfinishedRuns(ctx)
	if err != nil {
		return err
	}
	for _, run := range runs {
		if queue, ok := live[run.LectureID][run.RunID]; ok {
			current, exists := m.running[run.LectureID]
			if exists && current.queue != queue {
				return fmt.Errorf("lecture %d uses a different results queue", run.LectureID)
			}
			if !exists {
				consumerCtx, cancel := context.WithCancel(context.Background())
				current = resultConsumer{queue: queue, runs: make(map[string]struct{}), cancel: cancel}
				m.running[run.LectureID] = current
				go m.consume(consumerCtx, queue, run.LectureID)
			}
			current.runs[run.RunID] = struct{}{}
			continue
		}
		// A start that raced with the snapshot has not had time to appear in it.
		if run.StartedAt.After(checkedAt.Add(-runStartGrace)) {
			continue
		}
		if err := m.lectures.InterruptRun(ctx, run.LectureID, run.RunID); err != nil {
			return err
		}
		if current, ok := m.running[run.LectureID]; ok {
			delete(current.runs, run.RunID)
			if len(current.runs) == 0 {
				current.cancel()
				delete(m.running, run.LectureID)
			}
		}
		slog.Info("lecture_run_recovered", "lecture_id", run.LectureID, "run_id", run.RunID)
	}
	return nil
}

func inputQueueRunID(queue string) string {
	index := strings.LastIndexByte(queue, '.')
	if index < 0 {
		return ""
	}
	suffix := queue[index+1:]
	var id pgtype.UUID
	if id.Scan(suffix) == nil && id.Valid {
		return suffix
	}
	return ""
}

func (m *Manager) closeConsumers() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, current := range m.running {
		current.cancel()
		delete(m.running, id)
	}
}
