import { attempt, log, section } from './log';

type User = {
  readonly name: string;
};

type UserResponse = {
  readonly user: User;
};

/**
 * any は「型チェックをやめる」宣言。エディタは何も警告しないので、
 * 壊れたデータが来たことに気づくのが実行時になる。
 */
function greetWithAny(raw: any): string {
  return raw.user.name.toUpperCase();
}

/** unknown は使う前に確認を強制する。確認を書かないとコンパイルが通らない */
function greetWithUnknown(raw: unknown): string {
  if (!isUserResponse(raw)) return '(想定外のレスポンス。既定値で続行)';
  return raw.user.name.toUpperCase();
}

// as を使わずに絞り込む。in で確認した時点で TypeScript が型を狭めてくれる
function isUserResponse(value: unknown): value is UserResponse {
  if (typeof value !== 'object' || value === null) return false;
  if (!('user' in value)) return false;

  const user = value.user;
  if (typeof user !== 'object' || user === null) return false;
  if (!('name' in user)) return false;

  return typeof user.name === 'string';
}

export function run(): void {
  section('01. any と unknown');

  const valid = { user: { name: 'alice' } };
  const broken = { user: null };

  log('■ 正常なレスポンス', 'dim');
  attempt('any    ', () => greetWithAny(valid));
  attempt('unknown', () => greetWithUnknown(valid));

  log('', 'dim');
  log('■ サーバーの仕様変更で user が null になった', 'dim');
  attempt('any    ', () => greetWithAny(broken));
  attempt('unknown', () => greetWithUnknown(broken));

  log('', 'dim');
  log('any はここで初めて壊れる。unknown は壊れる前に気づける。', 'dim');
}
