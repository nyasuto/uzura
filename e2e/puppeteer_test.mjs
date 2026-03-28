// Puppeteer E2E test for Uzura CDP server.
// Usage: node puppeteer_test.mjs <ws-endpoint> <html-server-url>

import puppeteer from 'puppeteer-core';
import assert from 'node:assert/strict';

const wsEndpoint = process.argv[2];
const htmlURL = process.argv[3];

if (!wsEndpoint || !htmlURL) {
  console.error('Usage: node puppeteer_test.mjs <ws-endpoint> <html-url>');
  process.exit(1);
}

console.log(`Connecting to ${wsEndpoint}`);

const browser = await puppeteer.connect({ browserWSEndpoint: wsEndpoint });

try {
  const pages = await browser.pages();
  const page = pages[0];
  assert.ok(page, 'should have a page');

  // Test 1: Navigate to a page.
  console.log('Test 1: Navigate...');
  await page.goto(htmlURL, { waitUntil: 'load', timeout: 10000 });
  console.log('  PASS: navigation succeeded');

  // Test 2: Evaluate document.title.
  console.log('Test 2: evaluate document.title...');
  const title = await page.evaluate(() => document.title);
  assert.strictEqual(title, 'Test Page');
  console.log('  PASS: document.title matches');

  // Test 3: Evaluate querySelector + outerHTML.
  console.log('Test 3: evaluate querySelector + outerHTML...');
  const outerHTML = await page.evaluate(() => {
    const el = document.querySelector('h1');
    return el ? el.outerHTML : null;
  });
  assert.strictEqual(outerHTML, '<h1>Hello</h1>');
  console.log('  PASS: outerHTML matches');

  // Test 4: Evaluate querySelectorAll count.
  console.log('Test 4: evaluate querySelectorAll...');
  const pCount = await page.evaluate(() => {
    return document.querySelectorAll('p').length;
  });
  assert.strictEqual(pCount, 1); // <p>World</p>
  console.log('  PASS: querySelectorAll count matches');

  // Test 5: Evaluate textContent.
  console.log('Test 5: evaluate textContent...');
  const text = await page.evaluate(() => {
    const el = document.querySelector('.content p');
    return el ? el.textContent : null;
  });
  assert.strictEqual(text, 'World');
  console.log('  PASS: textContent matches');

  console.log('PASS: all puppeteer tests passed');
} finally {
  await browser.disconnect();
}
