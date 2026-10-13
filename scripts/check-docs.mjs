#!/usr/bin/env node
// ドキュメントの整合性チェック（依存なし / Node 20+）
//
//   node scripts/check-docs.mjs             … 相対リンク切れ・章IDの重複を検査
//   node scripts/check-docs.mjs --coverage  … 上に加え、docs/_source の全行が分割後の docs に残っているかを検査
//
// 失敗時は exit 1。PostToolUse フック（scripts/hooks/post-edit.sh）からも呼ばれる。

import { readFileSync, readdirSync, existsSync, statSync } from 'node:fs';
import { join, dirname, relative, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const DOCS = join(ROOT, 'docs');
const SOURCE = join(DOCS, '_source');

// リンク検査の対象（docs 配下とエージェント向けの指示ファイル）
const LINK_TARGET_DIRS = [DOCS, join(ROOT, '.claude')];
const LINK_TARGET_FILES = ['README.md', 'CLAUDE.md', 'AGENTS.md', 'app/CLAUDE.md', 'server/CLAUDE.md'].map((p) => join(ROOT, p));
// 章IDを持つディレクトリ
const ID_DIRS = [join(DOCS, 'product'), join(DOCS, 'tech')];

// 見出し先頭の章ID（例: "### DEAL-3 受け渡し…"、"## TECH-OPEN 未決事項"）
const HEADING_ID = /^(#{1,6}) ([A-Z]+(?:-[A-Z0-9]+)*) (?=\S)/;
// 元文書で失われた図のプレースホルダ（分割後は Mermaid で復元済み）
const EMBEDDED = /^&#91;embedded content/;

const errors = [];

function walk(dir, exclude = []) {
  if (!existsSync(dir)) return [];
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name);
    if (exclude.includes(p)) return [];
    if (statSync(p).isDirectory()) return walk(p, exclude);
    return name.endsWith('.md') ? [p] : [];
  });
}

const rel = (p) => relative(ROOT, p);
const normalize = (line) => line.trim().replace(HEADING_ID, '$1 ').replace(/\s+/g, ' ');

// --- 相対リンク切れ ---------------------------------------------------------
function checkLinks() {
  const files = [
    ...LINK_TARGET_DIRS.flatMap((d) => walk(d, [SOURCE])),
    ...LINK_TARGET_FILES.filter(existsSync),
  ];
  const linkRe = /\[[^\]]*\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
  for (const file of files) {
    let inFence = false;
    readFileSync(file, 'utf8').split('\n').forEach((line, i) => {
      if (/^\s*```/.test(line)) inFence = !inFence;
      if (inFence) return;
      for (const [, target] of line.matchAll(linkRe)) {
        if (/^(https?:|mailto:|#)/.test(target)) continue;
        const path = decodeURIComponent(target.split('#')[0]);
        if (!existsSync(resolve(dirname(file), path))) {
          errors.push(`リンク切れ: ${rel(file)}:${i + 1} → ${target}`);
        }
      }
    });
  }
}

// --- 章IDの重複 -------------------------------------------------------------
function checkIds() {
  const seen = new Map();
  for (const file of ID_DIRS.flatMap((d) => walk(d))) {
    readFileSync(file, 'utf8').split('\n').forEach((line, i) => {
      const m = line.match(HEADING_ID);
      if (!m) return;
      const where = `${rel(file)}:${i + 1}`;
      if (seen.has(m[2])) errors.push(`章IDの重複: ${m[2]}（${seen.get(m[2])} と ${where}）`);
      else seen.set(m[2], where);
    });
  }
}

// --- 元文書の行がすべて残っているか -----------------------------------------
function checkCoverage() {
  const migrated = new Set(
    walk(DOCS, [SOURCE]).flatMap((f) => readFileSync(f, 'utf8').split('\n').map(normalize)),
  );
  let total = 0;
  for (const file of walk(SOURCE)) {
    readFileSync(file, 'utf8').split('\n').forEach((line, i) => {
      const n = normalize(line);
      if (!n || EMBEDDED.test(n)) return;
      total++;
      if (!migrated.has(n)) errors.push(`取りこぼし: ${rel(file)}:${i + 1} ${n.slice(0, 60)}`);
    });
  }
  return total;
}

checkLinks();
checkIds();
const coverage = process.argv.includes('--coverage') ? checkCoverage() : null;

if (errors.length) {
  console.error(errors.join('\n'));
  console.error(`\ncheck-docs: ${errors.length} 件の問題があります`);
  process.exit(1);
}
console.log(
  coverage === null
    ? 'check-docs: OK（リンク・章ID）'
    : `check-docs: OK（リンク・章ID・元文書 ${coverage} 行すべて移行済み）`,
);
