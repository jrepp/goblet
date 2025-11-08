#!/usr/bin/env python3
"""
Load test harness for Goblet Git caching proxy.

This script simulates multiple concurrent Git clients making requests
to the proxy, testing cache efficiency, throughput, and stability.
"""

import argparse
import concurrent.futures
import hashlib
import json
import random
import statistics
import sys
import time
from dataclasses import dataclass
from typing import List, Dict, Tuple
import requests
from urllib.parse import urljoin


@dataclass
class TestResult:
    """Results from a single test request."""
    success: bool
    duration_ms: float
    repo: str
    operation: str
    served_by: str = ""
    error: str = ""


class GitProtocolV2Client:
    """Simple Git protocol v2 client for testing."""

    def __init__(self, base_url: str, timeout: int = 60):
        self.base_url = base_url
        self.timeout = timeout
        self.session = requests.Session()

    def ls_refs(self, repo_path: str) -> Tuple[bool, float, str, str]:
        """
        Execute ls-refs command.
        Returns: (success, duration_ms, served_by, error)
        """
        url = urljoin(self.base_url, f"/{repo_path}/git-upload-pack")

        # Git protocol v2 ls-refs payload
        payload = (
            b"0014command=ls-refs\n"
            b"0001"
            b"0009peel\n"
            b"000csymrefs\n"
            b"000bunborn\n"
            b"0014ref-prefix refs/\n"
            b"0000"
        )

        headers = {
            "Content-Type": "application/x-git-upload-pack-request",
            "Git-Protocol": "version=2",
            "Accept": "application/x-git-upload-pack-result",
        }

        start = time.time()
        try:
            response = self.session.post(
                url, data=payload, headers=headers, timeout=self.timeout
            )
            duration_ms = (time.time() - start) * 1000

            if response.status_code != 200:
                return False, duration_ms, "", f"HTTP {response.status_code}"

            if len(response.content) == 0:
                return False, duration_ms, "", "Empty response"

            served_by = response.headers.get("X-Served-By", "")
            return True, duration_ms, served_by, ""

        except Exception as e:
            duration_ms = (time.time() - start) * 1000
            return False, duration_ms, "", str(e)

    def fetch(self, repo_path: str, want_ref: str) -> Tuple[bool, float, str, str]:
        """
        Execute fetch command.
        Returns: (success, duration_ms, served_by, error)
        """
        url = urljoin(self.base_url, f"/{repo_path}/git-upload-pack")

        # Git protocol v2 fetch payload
        want_line = f"want {want_ref}\n".encode()
        payload = (
            b"0011command=fetch\n"
            b"0001"
            b"000cthin-pack\n"
            b"000cofs-delta\n"
            + f"{len(want_line) + 4:04x}".encode()
            + want_line
            + b"00000009done\n"
            b"0000"
        )

        headers = {
            "Content-Type": "application/x-git-upload-pack-request",
            "Git-Protocol": "version=2",
            "Accept": "application/x-git-upload-pack-result",
        }

        start = time.time()
        try:
            response = self.session.post(
                url, data=payload, headers=headers, timeout=self.timeout
            )
            duration_ms = (time.time() - start) * 1000

            if response.status_code != 200:
                return False, duration_ms, "", f"HTTP {response.status_code}"

            served_by = response.headers.get("X-Served-By", "")
            return True, duration_ms, served_by, ""

        except Exception as e:
            duration_ms = (time.time() - start) * 1000
            return False, duration_ms, "", str(e)


class LoadTestRunner:
    """Orchestrates load testing."""

    def __init__(
        self,
        target_url: str,
        repositories: List[str],
        num_workers: int = 10,
        requests_per_worker: int = 100,
        think_time_ms: int = 100,
    ):
        self.target_url = target_url
        self.repositories = repositories
        self.num_workers = num_workers
        self.requests_per_worker = requests_per_worker
        self.think_time_ms = think_time_ms
        self.results: List[TestResult] = []

    def worker_task(self, worker_id: int) -> List[TestResult]:
        """Worker function that executes requests."""
        client = GitProtocolV2Client(self.target_url)
        results = []

        for i in range(self.requests_per_worker):
            # Select random repository
            repo = random.choice(self.repositories)

            # 80% ls-refs, 20% fetch
            if random.random() < 0.8:
                success, duration, served_by, error = client.ls_refs(repo)
                result = TestResult(
                    success=success,
                    duration_ms=duration,
                    repo=repo,
                    operation="ls-refs",
                    served_by=served_by,
                    error=error,
                )
            else:
                # For fetch, we need a valid ref - use a common one
                # In real scenario, would ls-refs first
                dummy_ref = "0" * 40  # Placeholder
                success, duration, served_by, error = client.fetch(repo, dummy_ref)
                result = TestResult(
                    success=success,
                    duration_ms=duration,
                    repo=repo,
                    operation="fetch",
                    served_by=served_by,
                    error=error,
                )

            results.append(result)

            # Progress indicator
            if (i + 1) % 10 == 0:
                print(
                    f"Worker {worker_id}: {i + 1}/{self.requests_per_worker} requests",
                    end="\r",
                )

            # Think time
            if self.think_time_ms > 0:
                time.sleep(self.think_time_ms / 1000)

        return results

    def run(self) -> Dict:
        """Execute load test and return summary statistics."""
        print(f"Starting load test:")
        print(f"  Target: {self.target_url}")
        print(f"  Workers: {self.num_workers}")
        print(f"  Requests per worker: {self.requests_per_worker}")
        print(f"  Total requests: {self.num_workers * self.requests_per_worker}")
        print(f"  Repositories: {len(self.repositories)}")
        print()

        start_time = time.time()

        # Execute workers in parallel
        with concurrent.futures.ThreadPoolExecutor(
            max_workers=self.num_workers
        ) as executor:
            futures = [
                executor.submit(self.worker_task, i) for i in range(self.num_workers)
            ]

            for future in concurrent.futures.as_completed(futures):
                self.results.extend(future.result())

        total_duration = time.time() - start_time

        return self._compute_statistics(total_duration)

    def _compute_statistics(self, total_duration: float) -> Dict:
        """Compute summary statistics from results."""
        total_requests = len(self.results)
        successful = [r for r in self.results if r.success]
        failed = [r for r in self.results if not r.success]

        durations = [r.duration_ms for r in successful]

        # Server distribution
        server_counts = {}
        for r in self.results:
            if r.served_by:
                server_counts[r.served_by] = server_counts.get(r.served_by, 0) + 1

        # Repository distribution
        repo_requests = {}
        for r in self.results:
            repo_requests[r.repo] = repo_requests.get(r.repo, 0) + 1

        stats = {
            "total_requests": total_requests,
            "successful": len(successful),
            "failed": len(failed),
            "success_rate": len(successful) / total_requests * 100,
            "total_duration_sec": total_duration,
            "requests_per_sec": total_requests / total_duration,
            "duration_ms": {
                "min": min(durations) if durations else 0,
                "max": max(durations) if durations else 0,
                "mean": statistics.mean(durations) if durations else 0,
                "median": statistics.median(durations) if durations else 0,
                "p95": (
                    sorted(durations)[int(len(durations) * 0.95)]
                    if durations
                    else 0
                ),
                "p99": (
                    sorted(durations)[int(len(durations) * 0.99)]
                    if durations
                    else 0
                ),
            },
            "server_distribution": server_counts,
            "repo_distribution": repo_requests,
            "errors": {},
        }

        # Collect error types
        for r in failed:
            stats["errors"][r.error] = stats["errors"].get(r.error, 0) + 1

        return stats

    def print_summary(self, stats: Dict):
        """Print formatted summary statistics."""
        print("\n" + "=" * 60)
        print("LOAD TEST RESULTS")
        print("=" * 60)
        print(f"\nTotal Requests:    {stats['total_requests']}")
        print(f"Successful:        {stats['successful']}")
        print(f"Failed:            {stats['failed']}")
        print(f"Success Rate:      {stats['success_rate']:.2f}%")
        print(f"Total Duration:    {stats['total_duration_sec']:.2f}s")
        print(f"Requests/sec:      {stats['requests_per_sec']:.2f}")

        print(f"\nResponse Times (ms):")
        print(f"  Min:             {stats['duration_ms']['min']:.2f}")
        print(f"  Max:             {stats['duration_ms']['max']:.2f}")
        print(f"  Mean:            {stats['duration_ms']['mean']:.2f}")
        print(f"  Median:          {stats['duration_ms']['median']:.2f}")
        print(f"  P95:             {stats['duration_ms']['p95']:.2f}")
        print(f"  P99:             {stats['duration_ms']['p99']:.2f}")

        if stats["server_distribution"]:
            print(f"\nServer Distribution:")
            for server, count in sorted(stats["server_distribution"].items()):
                pct = count / stats["total_requests"] * 100
                print(f"  {server:20s} {count:6d} ({pct:5.2f}%)")

        if stats["errors"]:
            print(f"\nErrors:")
            for error, count in sorted(
                stats["errors"].items(), key=lambda x: x[1], reverse=True
            ):
                print(f"  {error:40s} {count:6d}")

        print("\n" + "=" * 60 + "\n")


def main():
    parser = argparse.ArgumentParser(
        description="Load test harness for Goblet Git caching proxy"
    )
    parser.add_argument(
        "--url",
        default="http://localhost:8080",
        help="Target URL (default: http://localhost:8080)",
    )
    parser.add_argument(
        "--workers", type=int, default=10, help="Number of concurrent workers"
    )
    parser.add_argument(
        "--requests", type=int, default=100, help="Requests per worker"
    )
    parser.add_argument(
        "--think-time", type=int, default=100, help="Think time between requests (ms)"
    )
    parser.add_argument(
        "--repos",
        nargs="+",
        default=[
            "github.com/kubernetes/kubernetes",
            "github.com/golang/go",
            "github.com/torvalds/linux",
            "github.com/hashicorp/terraform",
        ],
        help="List of repository paths to test",
    )
    parser.add_argument(
        "--output", help="Output file for JSON results (optional)"
    )

    args = parser.parse_args()

    runner = LoadTestRunner(
        target_url=args.url,
        repositories=args.repos,
        num_workers=args.workers,
        requests_per_worker=args.requests,
        think_time_ms=args.think_time,
    )

    stats = runner.run()
    runner.print_summary(stats)

    if args.output:
        with open(args.output, "w") as f:
            json.dump(stats, f, indent=2)
        print(f"Results saved to {args.output}")

    # Exit code based on success rate
    sys.exit(0 if stats["success_rate"] >= 95 else 1)


if __name__ == "__main__":
    main()
