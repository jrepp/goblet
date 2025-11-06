// Copyright 2025 Jacob Repp <jacobrepp@gmail.com>
//
// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oidc

import (
	"net/url"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CanonicalizeURL converts a proxy-style URL path to a canonical upstream Git URL.
// It supports paths like: /github.com/owner/repo, /gitlab.com/owner/repo, etc.
func CanonicalizeURL(u *url.URL) (*url.URL, error) {
	path := u.Path

	// Remove Git endpoint suffixes
	if strings.HasSuffix(path, "/info/refs") {
		path = strings.TrimSuffix(path, "/info/refs")
	} else if strings.HasSuffix(path, "/git-upload-pack") {
		path = strings.TrimSuffix(path, "/git-upload-pack")
	} else if strings.HasSuffix(path, "/git-receive-pack") {
		path = strings.TrimSuffix(path, "/git-receive-pack")
	}

	// Remove .git suffix
	path = strings.TrimSuffix(path, ".git")

	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		return nil, status.Error(codes.InvalidArgument, "empty repository path")
	}

	// Split path into host and repo path
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid repository path: %s (expected host/owner/repo)", path)
	}

	host := parts[0]
	repoPath := parts[1]

	// Validate host (basic check for domain format)
	if !strings.Contains(host, ".") {
		return nil, status.Errorf(codes.InvalidArgument, "invalid host: %s", host)
	}

	// Construct canonical URL
	canonical := &url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/" + repoPath,
	}

	return canonical, nil
}
