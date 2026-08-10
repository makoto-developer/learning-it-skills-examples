type Tone = 'plain' | 'ok' | 'ng' | 'dim';

export function log(text: string, _tone: Tone = 'plain'): void {
  console.log(text);
}

export function section(title: string): void {
  console.log('');
  console.log(`=== ${title} ===`);
}

/** 例外を投げるコードを「落ちた／落ちなかった」として観察するためのラッパ */
export function attempt(label: string, fn: () => string): void {
  try {
    console.log(`${label} -> ${fn()}`);
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    console.log(`${label} -> 実行時に落ちた: ${message}`);
  }
}
