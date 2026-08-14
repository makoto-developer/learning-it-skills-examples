/**
 * BFF（Backend For Frontend）。
 *
 * ブラウザは gRPC を直接話せないので、HTTP で受けて gRPC に変換する。
 * ここが「フロントに都合のよい形」に整える層でもある。
 */
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http';
import { Code, ConnectError, createClient, type Client } from '@connectrpc/connect';
import { createGrpcTransport } from '@connectrpc/connect-node';
import { LinkService } from './gen/link/v1/link_pb.js';

// 既定値は衝突しにくい番号にしてある。3000 や 8080 は他のツールと当たりやすい
const LINK_SERVICE_URL = process.env['LINK_SERVICE_URL'] ?? 'http://localhost:19001';
const PORT = Number(process.env['PORT'] ?? 19000);

const transport = createGrpcTransport({ baseUrl: LINK_SERVICE_URL });
const links: Client<typeof LinkService> = createClient(LinkService, transport);

type Json = Record<string, unknown>;

function send(res: ServerResponse, status: number, body: Json): void {
  const payload = JSON.stringify(body);
  res.writeHead(status, { 'content-type': 'application/json; charset=utf-8' });
  res.end(payload);
}

/** gRPC のステータスコードは HTTP のステータスと1対1ではないので、ここで翻訳する */
function toHttpStatus(error: unknown): number {
  if (!(error instanceof ConnectError)) return 500;
  switch (error.code) {
    case Code.InvalidArgument:
      return 400;
    case Code.NotFound:
      return 404;
    case Code.AlreadyExists:
      return 409;
    case Code.ResourceExhausted:
      return 429;
    case Code.Unavailable:
      return 503;
    default:
      return 500;
  }
}

function fail(res: ServerResponse, error: unknown): void {
  const status = toHttpStatus(error);
  const message = error instanceof ConnectError ? error.rawMessage : '内部エラーが発生しました';
  console.error(JSON.stringify({ level: 'error', status, message }));
  send(res, status, { error: message });
}

async function readJson(req: IncomingMessage): Promise<Json> {
  const chunks: Buffer[] = [];
  for await (const chunk of req) chunks.push(chunk as Buffer);
  if (chunks.length === 0) return {};
  return JSON.parse(Buffer.concat(chunks).toString('utf8')) as Json;
}

const server = createServer((req, res) => {
  void (async () => {
    const url = new URL(req.url ?? '/', `http://${req.headers.host ?? 'localhost'}`);

    try {
      // ヘルスチェック。Kubernetes の probe が叩く
      if (req.method === 'GET' && url.pathname === '/healthz') {
        send(res, 200, { status: 'ok' });
        return;
      }

      if (req.method === 'POST' && url.pathname === '/links') {
        const body = await readJson(req);
        const created = await links.createLink({ url: String(body['url'] ?? '') });
        send(res, 201, { key: created.link?.key, url: created.link?.url });
        return;
      }

      if (req.method === 'GET' && url.pathname === '/links') {
        const list = await links.listLinks({
          pageSize: Number(url.searchParams.get('page_size') ?? 0),
          pageToken: url.searchParams.get('page_token') ?? '',
        });
        send(res, 200, {
          links: list.links.map((link) => ({ key: link.key, url: link.url })),
          next_page_token: list.nextPageToken,
        });
        return;
      }

      // 短縮キーでのアクセスは、元の URL へ転送する
      const key = url.pathname.slice(1);
      if (req.method === 'GET' && key !== '') {
        const found = await links.getLink({ key });
        const target = found.link?.url;
        if (target === undefined) {
          send(res, 404, { error: 'そのキーは存在しません' });
          return;
        }
        res.writeHead(302, { location: target });
        res.end();
        return;
      }

      send(res, 404, { error: 'not found' });
    } catch (error) {
      fail(res, error);
    }
  })();
});

server.listen(PORT, () => {
  console.log(JSON.stringify({ level: 'info', msg: 'listening', port: PORT, upstream: LINK_SERVICE_URL }));
});

// SIGTERM を受けたら、処理中のリクエストを終わらせてから閉じる
for (const signal of ['SIGTERM', 'SIGINT'] as const) {
  process.on(signal, () => {
    console.log(JSON.stringify({ level: 'info', msg: 'shutting down' }));
    server.close(() => process.exit(0));
  });
}
