/*
 * Copyright (c) 2020-2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"context"
	"time"

	"github.com/devtron-labs/devtron/pkg/auth/user"

	"github.com/caarlos0/env"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

type TokenCache struct {
	cache           *cache.Cache
	logger          *zap.SugaredLogger
	aCDAuthConfig   *ACDAuthConfig
	userAuthService user.UserAuthService
}

func NewTokenCache(logger *zap.SugaredLogger, aCDAuthConfig *ACDAuthConfig, userAuthService user.UserAuthService) *TokenCache {
	tokenCache := &TokenCache{
		cache:           cache.New(cache.NoExpiration, 5*time.Minute),
		logger:          logger,
		aCDAuthConfig:   aCDAuthConfig,
		userAuthService: userAuthService,
	}
	return tokenCache
}
func (impl *TokenCache) BuildACDSynchContext() (acdContext context.Context, err error) {
	token, found := impl.cache.Get("token")
	impl.logger.Debugw("building acd context", "found", found)
	if !found {
		token, err := impl.userAuthService.HandleLogin(impl.aCDAuthConfig.ACDUsername, impl.aCDAuthConfig.ACDPassword)
		if err != nil {
			impl.logger.Errorw("error while acd login", "err", err)
			return nil, err
		}
		impl.cache.Set("token", token, cache.NoExpiration)
	}
	token, _ = impl.cache.Get("token")
	ctx := context.Background()
	ctx = context.WithValue(ctx, "token", token)
	return ctx, nil
}

// CATEGORY=GITOPS
type ACDAuthConfig struct {
	ACDUsername                      string `env:"ACD_USERNAME" envDefault:"admin" description:"User name for argocd"`
	ACDPassword                      string `env:"ACD_PASSWORD" description:"Password for the Argocd (deprecated)"`
	ACDConfigMapName                 string `env:"ACD_CM" envDefault:"argocd-cm" description:"Name of the argocd CM"`
	ACDConfigMapNamespace            string `env:"ACD_NAMESPACE" envDefault:"devtroncd" description:"To pass the argocd namespace"`
	GitOpsSecretName                 string `env:"GITOPS_SECRET_NAME" envDefault:"devtron-gitops-secret" description:"devtron-gitops-secret"`
	ResourceListForReplicas          string `env:"RESOURCE_LIST_FOR_REPLICAS" envDefault:"Deployment,Rollout,StatefulSet,ReplicaSet" description:"this holds the list of k8s resource names which support replicas key. this list used in hibernate/un hibernate process"`
	ResourceListForReplicasBatchSize int    `env:"RESOURCE_LIST_FOR_REPLICAS_BATCH_SIZE" envDefault:"5" description:"this the batch size to control no of above resources can be parsed in one go to determine hibernate status"`
}

func GetACDAuthConfig() (*ACDAuthConfig, error) {
	cfg := &ACDAuthConfig{}
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, err
}
