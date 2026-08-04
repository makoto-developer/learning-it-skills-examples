const output = document.getElementById('output');

type Tone = 'plain' | 'ok' | 'ng' | 'dim';

export function log(text: string, tone: Tone = 'plain'): void {
  console.log(text);
  if (output === null) return;

  const line = document.createElement('div');
  // innerHTML は使わない。文字列を DOM に流し込む癖をつけないため（第17章 XSS）
  line.textContent = text;
  if (tone !== 'plain') line.className = tone;
  output.appendChild(line);
}

export function section(title: string): void {
  console.log(`\n=== ${title} ===`);
  if (output === null) return;

  const heading = document.createElement('div');
  heading.textContent = title;
  heading.className = 'section';
  output.appendChild(heading);
}

/** 例外を投げるコードを「落ちた／落ちなかった」として観察するためのラッパ */
export function attempt(label: string, fn: () => string): void {
  try {
    log(`${label} -> ${fn()}`, 'ok');
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    log(`${label} -> 実行時に落ちた: ${message}`, 'ng');
  }
}
