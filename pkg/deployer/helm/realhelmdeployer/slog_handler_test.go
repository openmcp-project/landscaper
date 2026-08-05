// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package realhelmdeployer

import (
	"context"
	"log/slog"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openmcp-project/landscaper/controller-utils/pkg/logging"
)

var _ = Describe("messageBuffer", func() {

	Describe("add", func() {
		It("should add messages", func() {
			buf := &messageBuffer{}
			buf.add("a")
			buf.add("b")
			Expect(buf.messages).To(Equal([]string{"a", "b"}))
		})

		It("should deduplicate messages", func() {
			buf := &messageBuffer{}
			buf.add("a")
			buf.add("a")
			buf.add("b")
			buf.add("a")
			Expect(buf.messages).To(Equal([]string{"a", "b"}))
		})

		It("should keep only the last maxMessages messages", func() {
			buf := &messageBuffer{}
			for i := 0; i < maxMessages+3; i++ {
				buf.add(string(rune('a' + i)))
			}
			Expect(buf.messages).To(HaveLen(maxMessages))
			// the first 3 messages must have been dropped
			Expect(buf.messages[0]).To(Equal(string(rune('a' + 3))))
		})
	})

	Describe("get", func() {
		It("should return an empty string when no messages have been added", func() {
			buf := &messageBuffer{}
			Expect(buf.get()).To(Equal(""))
		})

		It("should return all messages each preceded by a newline", func() {
			buf := &messageBuffer{}
			buf.add("first")
			buf.add("second")
			Expect(buf.get()).To(Equal("\nfirst\nsecond"))
		})
	})
})

var _ = Describe("landscaperSlogHandler", func() {

	var ctx context.Context

	BeforeEach(func() {
		ctx = logging.NewContext(context.Background(), logging.Discard())
	})

	It("should accumulate messages in the buffer", func() {
		buf := &messageBuffer{}
		handler := newSlogHandler(ctx, buf)

		Expect(handler.Handle(ctx, slog.NewRecord(time.Time{}, slog.LevelInfo, "msg-1", 0))).To(Succeed())
		Expect(handler.Handle(ctx, slog.NewRecord(time.Time{}, slog.LevelInfo, "msg-2", 0))).To(Succeed())

		Expect(buf.get()).To(Equal("\nmsg-1\nmsg-2"))
	})

	It("should not panic when buf is nil", func() {
		handler := newSlogHandler(ctx, nil)
		Expect(func() {
			_ = handler.Handle(ctx, slog.NewRecord(time.Time{}, slog.LevelInfo, "msg", 0))
		}).NotTo(Panic())
	})

	It("should always report as enabled regardless of level", func() {
		handler := newSlogHandler(ctx, nil)
		Expect(handler.Enabled(ctx, slog.LevelDebug)).To(BeTrue())
		Expect(handler.Enabled(ctx, slog.LevelError)).To(BeTrue())
	})

	It("should propagate the buffer through WithAttrs", func() {
		buf := &messageBuffer{}
		handler := newSlogHandler(ctx, buf)
		child := handler.WithAttrs([]slog.Attr{slog.String("k", "v")})
		Expect(child.Handle(ctx, slog.NewRecord(time.Time{}, slog.LevelInfo, "from-child", 0))).To(Succeed())
		Expect(buf.get()).To(ContainSubstring("from-child"))
	})

	It("should propagate the buffer through WithGroup", func() {
		buf := &messageBuffer{}
		handler := newSlogHandler(ctx, buf)
		child := handler.WithGroup("mygroup")
		Expect(child.Handle(ctx, slog.NewRecord(time.Time{}, slog.LevelInfo, "from-group", 0))).To(Succeed())
		Expect(buf.get()).To(ContainSubstring("from-group"))
	})
})
