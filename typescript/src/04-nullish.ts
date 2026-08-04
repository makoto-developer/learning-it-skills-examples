import { log, section } from './log';

type Settings = {
  readonly pageSize?: number;
  readonly nickname?: string;
  readonly notify?: boolean;
};

export function run(): void {
  section('04. || と ?? の違い（0 が消えるバグ）');

  const inputs: readonly Settings[] = [
    { pageSize: 50, nickname: 'alice', notify: true },
    { pageSize: 0, nickname: '', notify: false },
    {},
  ];

  for (const settings of inputs) {
    log(`■ 入力: ${JSON.stringify(settings)}`, 'dim');

    const sizeOr = settings.pageSize || 10;
    const sizeNullish = settings.pageSize ?? 10;
    log(`  pageSize  || 10 -> ${sizeOr}`, sizeOr === sizeNullish ? 'plain' : 'ng');
    log(`  pageSize  ?? 10 -> ${sizeNullish}`, 'ok');

    const nameOr = settings.nickname || '名無し';
    const nameNullish = settings.nickname ?? '名無し';
    log(`  nickname  || '名無し' -> ${JSON.stringify(nameOr)}`, nameOr === nameNullish ? 'plain' : 'ng');
    log(`  nickname  ?? '名無し' -> ${JSON.stringify(nameNullish)}`, 'ok');

    const notifyOr = settings.notify || true;
    const notifyNullish = settings.notify ?? true;
    log(`  notify    || true -> ${notifyOr}`, notifyOr === notifyNullish ? 'plain' : 'ng');
    log(`  notify    ?? true -> ${notifyNullish}`, 'ok');
  }

  log('', 'dim');
  log('2 件目に注目。|| は 0 / 空文字 / false も「無い」とみなして既定値に置き換える。', 'dim');
  log('「0 を指定できない」「通知オフにしたのに戻る」はこれが原因。', 'dim');
}
