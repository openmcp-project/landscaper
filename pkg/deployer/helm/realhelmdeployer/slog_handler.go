// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"context"
	"log/slog"
	"sync"

	"github.com/openmcp-project/landscaper/controller-utils/pkg/logging"
	lc "github.com/openmcp-project/landscaper/controller-utils/pkg/logging/constants"
)

const maxMessages = 10

// messageBuffer holds the last maxMessages unique log messages, thread-safely.
type messageBuffer struct {
	mu       sync.RWMutex
	messages []string
}

func (b *messageBuffer) add(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, m := range b.messages {
		if m == msg {
			return
		}
	}
	b.messages = append(b.messages, msg)
	if len(b.messages) > maxMessages {
		b.messages = b.messages[len(b.messages)-maxMessages:]
	}
}

func (b *messageBuffer) get() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := ""
	for _, m := range b.messages {
		result += "\n" + m
	}
	return result
}

// newSlogHandler returns a slog.Handler that forwards log records to the landscaper logger from ctx
// and accumulates them in buf.
func newSlogHandler(ctx context.Context, buf *messageBuffer) slog.Handler {
	logger, _ := logging.FromContextOrNew(ctx, []interface{}{lc.KeyMethod, "RealHelmDeployer"})
	return &landscaperSlogHandler{logger: logger, buf: buf}
}

// landscaperSlogHandler bridges helm's slog-based logging to the landscaper logger
// and keeps a deduplicated rolling window of the last maxMessages messages.
// It serves a twofold purpose:
//   - It forwards all helm log records to the landscaper logger so they appear in the operator logs.
//   - It accumulates the messages in buf (if non-nil), so they can be included in the deploy item's
//     error status when an install or upgrade fails.
type landscaperSlogHandler struct {
	logger logging.Logger
	buf    *messageBuffer
	attrs  []slog.Attr
	group  string
}

func (h *landscaperSlogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *landscaperSlogHandler) Handle(_ context.Context, r slog.Record) error {
	if h.buf != nil {
		h.buf.add(r.Message)
	}
	h.logger.Info(r.Message)
	return nil
}

func (h *landscaperSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &landscaperSlogHandler{logger: h.logger, buf: h.buf, attrs: append(h.attrs, attrs...), group: h.group}
}

func (h *landscaperSlogHandler) WithGroup(name string) slog.Handler {
	return &landscaperSlogHandler{logger: h.logger, buf: h.buf, attrs: h.attrs, group: name}
}
