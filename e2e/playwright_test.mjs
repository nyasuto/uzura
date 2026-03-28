// Playwright E2E test for Uzura CDP server (CDP mode).
// Usage: node playwright_test.mjs <ws-endpoint> <html-server-url>

import { chromium } from 'playwright-core';
import assert from 'node:assert/strict';

const wsEndpoint = process.argv[2];
const htmlURL = process.argv[3];

if (!wsEndpoint || !htmlURL) {
  console.error('Usage: node playwright_test.mjs <ws-endpoint> <html-url>');
  process.exit(1);
}

console.log(`Connecting to ${wsEndpoint} via Playwright CDP`);

const browser = await chromium.connectOverCDP(wsEndpoint);

try {
  // Test 1: Verify connection and page availability.
  console.log('Test 1: Connection and page discovery...');
  const context = browser.contexts()[0];
  assert.ok(context, 'should have a browser context');
  const page = context.pages()[0];
  assert.ok(page, 'should have a page');
  console.log('  PASS: connected with context and page');

  // Test 2: Browser version via CDP.
  console.log('Test 2: Browser version...');
  const version = browser.version();
  console.log(`  INFO: browser.version() = ${JSON.stringify(version)}`);
  assert.ok(typeof version === 'string', 'version should be a string');
  console.log(`  PASS: browser version is a string`);

  // Test 3: URL of the current page.
  console.log('Test 3: Current page URL...');
  const url = page.url();
  assert.ok(url === '' || url === 'about:blank', 'initial URL should be about:blank or empty');
  console.log(`  PASS: initial URL = ${url}`);

  console.log('PASS: all playwright tests passed');
} finally {
  await browser.close();
}
