import { log, section } from './log';

/**
 * 状態を「フラグの組み合わせ」ではなく1つのユニオンで表す。
 * ありえない組み合わせ（loading なのに data がある等）を型で作れなくする。
 */
type FetchState =
  | { readonly status: 'loading' }
  | { readonly status: 'success'; readonly data: string }
  | { readonly status: 'error'; readonly message: string };

function render(state: FetchState): string {
  switch (state.status) {
    case 'loading':
      return '読み込み中...';
    case 'success':
      return `成功: ${state.data}`;
    case 'error':
      return `エラー: ${state.message}`;
    default:
      return assertNever(state);
  }
}

/**
 * ここに到達しうる型が残っていればコンパイルエラーになる。
 * FetchState に状態を1つ足して、この行が赤くなるのを確かめてほしい。
 */
function assertNever(value: never): never {
  throw new Error(`未処理のケース: ${JSON.stringify(value)}`);
}

export function run(): void {
  section('03. 判別可能ユニオンと網羅性チェック');

  const states: readonly FetchState[] = [
    { status: 'loading' },
    { status: 'success', data: 'ユーザー3件' },
    { status: 'error', message: 'タイムアウト' },
  ];

  for (const state of states) {
    log(`${state.status.padEnd(8)} -> ${render(state)}`, 'ok');
  }

  log('', 'dim');
  log("FetchState に { status: 'empty' } を足すと、render の default が", 'dim');
  log('コンパイルエラーになる。対応漏れを実行前に気づける。', 'dim');
}
