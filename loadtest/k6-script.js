import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const cacheHitRate = new Rate('cache_hits');
const requestDuration = new Trend('request_duration');
const requestCounter = new Counter('requests_total');

// Load test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 VUs
    { duration: '5m', target: 10 },   // Stay at 10 VUs
    { duration: '2m', target: 50 },   // Ramp up to 50 VUs
    { duration: '5m', target: 50 },   // Stay at 50 VUs
    { duration: '2m', target: 100 },  // Ramp up to 100 VUs
    { duration: '5m', target: 100 },  // Stay at 100 VUs
    { duration: '2m', target: 0 },    // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<5000'], // 95% of requests should be below 5s
    'errors': ['rate<0.1'],               // Error rate should be below 10%
    'http_req_failed': ['rate<0.05'],     // Failed requests below 5%
  },
};

// Simulated repository list (adjust to match your test repos)
const repositories = [
  'github.com/kubernetes/kubernetes',
  'github.com/golang/go',
  'github.com/torvalds/linux',
  'github.com/facebook/react',
  'github.com/microsoft/vscode',
  'github.com/hashicorp/terraform',
  'github.com/nodejs/node',
  'github.com/rust-lang/rust',
  'github.com/apache/spark',
  'github.com/tensorflow/tensorflow',
];

// Git protocol v2 ls-refs command
function createLsRefsRequest() {
  return '0014command=ls-refs\n' +
         '0001' +
         '0009peel\n' +
         '000csymrefs\n' +
         '000bunborn\n' +
         '0014ref-prefix refs/\n' +
         '0000';
}

// Git protocol v2 fetch command (minimal)
function createFetchRequest(wantRef) {
  return '0011command=fetch\n' +
         '0001' +
         '000cthin-pack\n' +
         '000cofs-delta\n' +
         `00${(32 + wantRef.length).toString(16).padStart(2, '0')}want ${wantRef}\n` +
         '00000009done\n' +
         '0000';
}

// Select random repository
function getRandomRepo() {
  return repositories[Math.floor(Math.random() * repositories.length)];
}

export default function () {
  const targetUrl = __ENV.TARGET_URL || 'http://localhost:8080';
  const repo = getRandomRepo();
  const repoUrl = `${targetUrl}/${repo}/git-upload-pack`;

  // Scenario 1: ls-refs request (80% of requests)
  if (Math.random() < 0.8) {
    const lsRefsPayload = createLsRefsRequest();

    const params = {
      headers: {
        'Content-Type': 'application/x-git-upload-pack-request',
        'Git-Protocol': 'version=2',
        'Accept': 'application/x-git-upload-pack-result',
      },
      timeout: '60s',
    };

    const start = Date.now();
    const response = http.post(repoUrl, lsRefsPayload, params);
    const duration = Date.now() - start;

    requestCounter.add(1);
    requestDuration.add(duration);

    const success = check(response, {
      'ls-refs status is 200': (r) => r.status === 200,
      'ls-refs has body': (r) => r.body.length > 0,
      'ls-refs is valid': (r) => r.body.includes('refs/'),
    });

    errorRate.add(!success);

    // Check if served from cache (custom header from HAProxy)
    if (response.headers['X-Served-By']) {
      console.log(`Repo ${repo} served by ${response.headers['X-Served-By']}`);
    }
  }
  // Scenario 2: fetch request (20% of requests)
  else {
    // First, get refs with ls-refs
    const lsRefsPayload = createLsRefsRequest();
    const params = {
      headers: {
        'Content-Type': 'application/x-git-upload-pack-request',
        'Git-Protocol': 'version=2',
        'Accept': 'application/x-git-upload-pack-result',
      },
      timeout: '60s',
    };

    const lsRefsResponse = http.post(repoUrl, lsRefsPayload, params);

    if (lsRefsResponse.status === 200) {
      // Parse a ref from response (simplified - assumes valid format)
      const refMatch = lsRefsResponse.body.match(/([0-9a-f]{40})\s+refs\/heads\/\w+/);

      if (refMatch && refMatch[1]) {
        const wantRef = refMatch[1];
        const fetchPayload = createFetchRequest(wantRef);

        const start = Date.now();
        const fetchResponse = http.post(repoUrl, fetchPayload, params);
        const duration = Date.now() - start;

        requestCounter.add(1);
        requestDuration.add(duration);

        const success = check(fetchResponse, {
          'fetch status is 200': (r) => r.status === 200,
          'fetch has pack data': (r) => r.body.length > 0,
        });

        errorRate.add(!success);
      }
    }
  }

  // Think time between requests (simulates real user behavior)
  sleep(Math.random() * 3 + 1); // 1-4 seconds
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    '/tmp/k6-summary.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = '\n' + indent + '=== Load Test Summary ===\n\n';

  summary += indent + `Requests: ${data.metrics.requests_total.values.count}\n`;
  summary += indent + `Errors: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%\n`;
  summary += indent + `Request Duration (p95): ${data.metrics.request_duration.values['p(95)']}ms\n`;
  summary += indent + `HTTP Req Duration (p95): ${data.metrics.http_req_duration.values['p(95)']}ms\n`;

  return summary;
}
