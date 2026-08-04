import { z } from 'zod';
import { log, section } from './log';

const userSchema = z.object({
  id: z.string(),
  name: z.string(),
  age: z.number().int().positive(),
  email: z.string().email(),
});

type User = z.infer<typeof userSchema>;

/** 境界で1回検証する。ここを通過したあとは型を信じてよい */
function parseUser(raw: unknown): User | undefined {
  const result = userSchema.safeParse(raw);
  if (!result.success) {
    for (const issue of result.error.issues) {
      const path = issue.path.length === 0 ? '(ルート)' : issue.path.join('.');
      log(`  NG ${path}: ${issue.message}`, 'ng');
    }
    return undefined;
  }
  return result.data;
}

export function run(): void {
  section('02. 外部から来るデータを実行時に検証する');

  const responses: readonly unknown[] = [
    { id: 'u1', name: 'alice', age: 24, email: 'alice@example.com' },
    { id: 'u2', name: 'bob', age: -3, email: 'bob@example.com' },
    { id: 'u3', name: 'carol', age: 30, email: 'not-an-email' },
    { id: 999, name: 'dave', age: 41 },
  ];

  for (const [index, raw] of responses.entries()) {
    log(`■ ${index + 1} 件目: ${JSON.stringify(raw)}`, 'dim');
    const user = parseUser(raw);
    if (user !== undefined) log(`  OK ${user.name} (${user.age}歳)`, 'ok');
  }

  log('', 'dim');
  log('型注釈だけなら 2〜4 件目も素通りし、後続の処理で謎の値として現れる。', 'dim');
}
