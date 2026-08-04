import { log, section } from './log';

function delay(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchUser(id: string): Promise<string> {
  await delay(300);
  if (id === 'ng') throw new Error(`ユーザー ${id} の取得に失敗`);
  return `user:${id}`;
}

async function measure(label: string, fn: () => Promise<void>): Promise<void> {
  const start = performance.now();
  await fn();
  log(`  ${label}: ${Math.round(performance.now() - start)}ms`, 'dim');
}

export async function run(): Promise<void> {
  section('05. 非同期 — 直列と並列、all と allSettled');

  const ids = ['a', 'b', 'c'];

  log('■ 待つ必要のないものを直列で待つと、その分だけ遅くなる', 'dim');
  await measure('直列 (for await)', async () => {
    for (const id of ids) await fetchUser(id);
  });
  await measure('並列 (Promise.all)', async () => {
    await Promise.all(ids.map(fetchUser));
  });

  log('', 'dim');
  log('■ 1件失敗したときの違い', 'dim');

  try {
    await Promise.all(['a', 'ng', 'c'].map(fetchUser));
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    log(`  Promise.all        -> 全体が失敗する: ${message}`, 'ng');
  }

  const settled = await Promise.allSettled(['a', 'ng', 'c'].map(fetchUser));
  for (const result of settled) {
    if (result.status === 'fulfilled') log(`  allSettled fulfilled -> ${result.value}`, 'ok');
    else log(`  allSettled rejected  -> ${String(result.reason)}`, 'ng');
  }

  log('', 'dim');
  log('全部揃わないと意味がないなら all、1件ずつ結果を扱いたいなら allSettled。', 'dim');
}
