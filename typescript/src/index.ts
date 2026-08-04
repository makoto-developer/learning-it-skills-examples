import { log } from './log';
import { run as runUnknown } from './01-unknown';
import { run as runValidation } from './02-validation';
import { run as runUnion } from './03-discriminated-union';
import { run as runNullish } from './04-nullish';
import { run as runAsync } from './05-async';

async function main(): Promise<void> {
  runUnknown();
  runValidation();
  runUnion();
  runNullish();
  await runAsync();

  log('', 'dim');
  log('--- 以上 5 本 ---', 'dim');
}

void main();
