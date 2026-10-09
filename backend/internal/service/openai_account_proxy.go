package service

import (
	"context"
	"fmt"
)

func resolveOpenAIAccountProxy(ctx context.Context, account *Account, repo ProxyRepository) (string, error) {
	if account.ProxyID == nil {
		return "", nil
	}
	if account.Proxy != nil {
		return account.Proxy.URL(), nil
	}
	if repo == nil {
		return "", fmt.Errorf("OpenAI account %d has a bound proxy that was not loaded", account.ID)
	}
	proxy, err := repo.GetByID(ctx, *account.ProxyID)
	if err != nil {
		return "", fmt.Errorf("resolve OpenAI account %d proxy: %w", account.ID, err)
	}
	if proxy == nil {
		return "", fmt.Errorf("OpenAI account %d bound proxy %d was not found", account.ID, *account.ProxyID)
	}
	return proxy.URL(), nil
}
