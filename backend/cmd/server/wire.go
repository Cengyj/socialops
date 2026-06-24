//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Wei-Shaw/socialops/ent"
	"github.com/Wei-Shaw/socialops/internal/config"
	"github.com/Wei-Shaw/socialops/internal/handler"
	"github.com/Wei-Shaw/socialops/internal/payment"
	"github.com/Wei-Shaw/socialops/internal/repository"
	"github.com/Wei-Shaw/socialops/internal/server"
	"github.com/Wei-Shaw/socialops/internal/server/middleware"
	"github.com/Wei-Shaw/socialops/internal/service"

	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
)

type Application struct {
	Server  *http.Server
	Cleanup func()
}

func initializeApplication(buildInfo handler.BuildInfo) (*Application, error) {
	wire.Build(
		config.ProviderSet,
		repository.ProviderSet,
		service.ProviderSet,
		payment.ProviderSet,
		middleware.ProviderSet,
		handler.ProviderSet,
		server.ProviderSet,
		provideServiceBuildInfo,
		provideCleanup,
		wire.Struct(new(Application), "Server", "Cleanup"),
	)
	return nil, nil
}

func provideServiceBuildInfo(buildInfo handler.BuildInfo) service.BuildInfo {
	return service.BuildInfo{
		Version:   buildInfo.Version,
		BuildType: buildInfo.BuildType,
	}
}

func provideCleanup(
	entClient *ent.Client,
	rdb *redis.Client,
	emailQueue *service.EmailQueueService,
	billingCache *service.BillingCacheService,
	subscriptionService *service.SubscriptionService,
	backupSvc *service.BackupService,
	paymentOrderExpiry *service.PaymentOrderExpiryService,
	subscriptionExpiry *service.SubscriptionExpiryService,
	_ *service.IdempotencyCoordinator,
	socialTaskExecutor *service.SocialTaskExecutor,
) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		type cleanupStep struct {
			name string
			fn   func() error
		}

		parallelSteps := []cleanupStep{
			{"SocialTaskExecutor", func() error {
				if socialTaskExecutor != nil {
					socialTaskExecutor.Stop()
				}
				return nil
			}},
			{"SubscriptionExpiryService", func() error {
				if subscriptionExpiry != nil {
					subscriptionExpiry.Stop()
				}
				return nil
			}},
			{"SubscriptionService", func() error {
				if subscriptionService != nil {
					subscriptionService.Stop()
				}
				return nil
			}},
			{"EmailQueueService", func() error {
				if emailQueue != nil {
					emailQueue.Stop()
				}
				return nil
			}},
			{"BillingCacheService", func() error {
				if billingCache != nil {
					billingCache.Stop()
				}
				return nil
			}},
			{"BackupService", func() error {
				if backupSvc != nil {
					backupSvc.Stop()
				}
				return nil
			}},
			{"PaymentOrderExpiryService", func() error {
				if paymentOrderExpiry != nil {
					paymentOrderExpiry.Stop()
				}
				return nil
			}},
		}

		infraSteps := []cleanupStep{
			{"Redis", func() error {
				if rdb == nil {
					return nil
				}
				return rdb.Close()
			}},
			{"Ent", func() error {
				if entClient == nil {
					return nil
				}
				return entClient.Close()
			}},
		}

		runParallel := func(steps []cleanupStep) {
			var wg sync.WaitGroup
			for i := range steps {
				step := steps[i]
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := step.fn(); err != nil {
						log.Printf("[Cleanup] %s failed: %v", step.name, err)
						return
					}
					log.Printf("[Cleanup] %s succeeded", step.name)
				}()
			}
			wg.Wait()
		}

		runSequential := func(steps []cleanupStep) {
			for i := range steps {
				step := steps[i]
				if err := step.fn(); err != nil {
					log.Printf("[Cleanup] %s failed: %v", step.name, err)
					continue
				}
				log.Printf("[Cleanup] %s succeeded", step.name)
			}
		}

		runParallel(parallelSteps)
		runSequential(infraSteps)

		select {
		case <-ctx.Done():
			log.Printf("[Cleanup] Warning: cleanup timed out after 10 seconds")
		default:
			log.Printf("[Cleanup] All cleanup steps completed")
		}
	}
}
