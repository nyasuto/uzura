#!/usr/bin/env node
// bench-compare.mjs — Compare Uzura vs Headless Chrome performance.
//
// Usage:
//   node scripts/bench-compare.mjs [--uzura-port 9222] [--chrome-port 9223] [--iterations 5]
//
// Prerequisites:
//   - Uzura running: ./uzura serve --port 9222
//   - Chrome running: google-chrome --headless --remote-debugging-port=9223
//   - npm install in e2e/ directory

import puppeteer from 'puppeteer-core';

const args = process.argv.slice(2);
function getArg(name, def) {
  const idx = args.indexOf(`--${name}`);
  return idx >= 0 && args[idx + 1] ? args[idx + 1] : def;
}

const UZURA_PORT = parseInt(getArg('uzura-port', '9222'));
const CHROME_PORT = parseInt(getArg('chrome-port', '9223'));
const ITERATIONS = parseInt(getArg('iterations', '5'));

const HTML_SMALL = `data:text/html,<!DOCTYPE html><html><body>${Array.from({length: 10}, (_, i) => `<div class="item" id="item-${i}"><p>Item ${i}</p></div>`).join('')}</body></html>`;
const HTML_LARGE = `data:text/html,<!DOCTYPE html><html><body>${Array.from({length: 500}, (_, i) => `<div class="item" id="item-${i}" data-idx="${i}"><h3>Item ${i}</h3><p>Desc ${i}</p></div>`).join('')}</body></html>`;

async function connectBrowser(name, port) {
  try {
    const browser = await puppeteer.connect({
      browserWSEndpoint: `ws://127.0.0.1:${port}/devtools/browser`,
    });
    return browser;
  } catch {
    console.error(`Could not connect to ${name} on port ${port}. Skipping.`);
    return null;
  }
}

async function benchmark(page, name, fn) {
  // Warmup
  await fn(page);

  const times = [];
  for (let i = 0; i < ITERATIONS; i++) {
    const start = performance.now();
    await fn(page);
    times.push(performance.now() - start);
  }

  const avg = times.reduce((a, b) => a + b, 0) / times.length;
  const min = Math.min(...times);
  const max = Math.max(...times);
  return { name, avg, min, max };
}

const scenarios = [
  {
    name: 'Navigate small page',
    fn: async (page) => {
      await page.goto(HTML_SMALL, { waitUntil: 'load' });
    },
  },
  {
    name: 'Navigate large page',
    fn: async (page) => {
      await page.goto(HTML_LARGE, { waitUntil: 'load' });
    },
  },
  {
    name: 'querySelector (large DOM)',
    fn: async (page) => {
      await page.goto(HTML_LARGE, { waitUntil: 'load' });
      await page.evaluate(() => document.querySelector('#item-250'));
    },
  },
  {
    name: 'querySelectorAll (large DOM)',
    fn: async (page) => {
      await page.goto(HTML_LARGE, { waitUntil: 'load' });
      await page.evaluate(() => document.querySelectorAll('.item').length);
    },
  },
  {
    name: 'DOM mutation',
    fn: async (page) => {
      await page.goto(HTML_SMALL, { waitUntil: 'load' });
      await page.evaluate(() => {
        for (let i = 0; i < 100; i++) {
          const el = document.createElement('div');
          el.textContent = `new-${i}`;
          document.body.appendChild(el);
        }
      });
    },
  },
  {
    name: 'JS evaluation',
    fn: async (page) => {
      await page.evaluate(() => {
        let sum = 0;
        for (let i = 0; i < 1000; i++) sum += i;
        return sum;
      });
    },
  },
];

async function runSuite(browser, label) {
  const page = await browser.newPage();
  const results = [];
  for (const scenario of scenarios) {
    try {
      const result = await benchmark(page, scenario.name, scenario.fn);
      results.push(result);
    } catch (e) {
      results.push({ name: scenario.name, avg: NaN, min: NaN, max: NaN, error: e.message });
    }
  }
  await page.close();
  return results;
}

function formatMs(ms) {
  if (isNaN(ms)) return 'ERROR';
  if (ms < 1) return `${(ms * 1000).toFixed(0)} μs`;
  return `${ms.toFixed(2)} ms`;
}

async function main() {
  console.log(`Benchmark: ${ITERATIONS} iterations per scenario\n`);

  const uzura = await connectBrowser('Uzura', UZURA_PORT);
  const chrome = await connectBrowser('Chrome', CHROME_PORT);

  if (!uzura && !chrome) {
    console.error('No browsers available. Start at least one.');
    process.exit(1);
  }

  let uzuraResults = null;
  let chromeResults = null;

  if (uzura) {
    console.log('Running Uzura benchmarks...');
    uzuraResults = await runSuite(uzura, 'Uzura');
    await uzura.disconnect();
  }

  if (chrome) {
    console.log('Running Chrome benchmarks...');
    chromeResults = await runSuite(chrome, 'Chrome');
    await chrome.disconnect();
  }

  // Output as Markdown table.
  console.log('\n## CDP Benchmark Comparison\n');

  const headers = ['Scenario'];
  if (uzuraResults) headers.push('Uzura (avg)', 'Uzura (min)');
  if (chromeResults) headers.push('Chrome (avg)', 'Chrome (min)');
  if (uzuraResults && chromeResults) headers.push('Ratio');

  console.log(`| ${headers.join(' | ')} |`);
  console.log(`|${headers.map(() => ':--').join('|')}|`);

  for (let i = 0; i < scenarios.length; i++) {
    const cols = [scenarios[i].name];
    if (uzuraResults) {
      cols.push(formatMs(uzuraResults[i].avg), formatMs(uzuraResults[i].min));
    }
    if (chromeResults) {
      cols.push(formatMs(chromeResults[i].avg), formatMs(chromeResults[i].min));
    }
    if (uzuraResults && chromeResults) {
      const ratio = uzuraResults[i].avg / chromeResults[i].avg;
      cols.push(isNaN(ratio) ? 'N/A' : `${ratio.toFixed(2)}x`);
    }
    console.log(`| ${cols.join(' | ')} |`);
  }
  console.log('');

  // JSON output for CI.
  const jsonOutput = {
    iterations: ITERATIONS,
    uzura: uzuraResults,
    chrome: chromeResults,
  };
  process.stdout.write(`<!-- bench-compare-json\n${JSON.stringify(jsonOutput, null, 2)}\n-->\n`);
}

main().catch(e => {
  console.error(e);
  process.exit(1);
});
