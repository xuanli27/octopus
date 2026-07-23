package relay

import (
	"context"
	"errors"
	"strings"

	"github.com/xuanli27/octopus/internal/transformer/model"
)

var (
	errLocalRelayBudgetExceeded = errors.New("local relay budget exceeded")
	errFirstTokenTimeout        = errors.New("first token timeout")
)

func contextError(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return ctx.Err()
}

func isLocalRelayBudgetExceeded(ctx context.Context, err error) bool {
	if errors.Is(err, errLocalRelayBudgetExceeded) {
		return true
	}
	if ctx == nil {
		return false
	}
	return errors.Is(context.Cause(ctx), errLocalRelayBudgetExceeded)
}

func isFirstTokenTimeout(ctx context.Context, err error) bool {
	if errors.Is(err, errFirstTokenTimeout) {
		return true
	}
	if ctx == nil {
		return false
	}
	return errors.Is(context.Cause(ctx), errFirstTokenTimeout)
}

func isClientCancellation(ctx context.Context, err error) bool {
	if isLocalRelayBudgetExceeded(ctx, err) || isLocalRelayBudgetExceeded(ctx, contextError(ctx)) ||
		isFirstTokenTimeout(ctx, err) || isFirstTokenTimeout(ctx, contextError(ctx)) {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if ctx == nil {
		return false
	}
	return errors.Is(ctx.Err(), context.Canceled) || errors.Is(ctx.Err(), context.DeadlineExceeded)
}

// streamResponseCompleted reports whether an aggregated Chat-style stream response
// already reached a terminal finish_reason on every choice.
//
// Used to normalize late client disconnects (context canceled after the full
// answer was delivered) into success, instead of recording a false failure.
// See GitHub issues #116 / #111.
func streamResponseCompleted(resp *model.InternalLLMResponse) bool {
	if resp == nil || len(resp.Choices) == 0 {
		return false
	}
	for i := range resp.Choices {
		fr := resp.Choices[i].FinishReason
		if fr == nil || strings.TrimSpace(*fr) == "" {
			return false
		}
	}
	return true
}
