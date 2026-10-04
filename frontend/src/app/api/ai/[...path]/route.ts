import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:8000';

function sanitizePath(segments: string[]): string | null {
  if (!segments || segments.length === 0) return null;
  for (const seg of segments) {
    if (!seg || seg === '.' || seg === '..' || seg.includes('/') || seg.includes('\\')) {
      return null;
    }
  }
  return segments.join('/');
}

export async function GET(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  const safePath = sanitizePath(path);
  if (!safePath) {
    return NextResponse.json({ error: 'Invalid path' }, { status: 400 });
  }

  const search = request.nextUrl.search || '';
  try {
    const response = await fetch(`${BACKEND_URL}/ai/${safePath}${search}`, {
      headers: { 'Content-Type': 'application/json' },
    });
    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch {
    return NextResponse.json({ error: 'Failed to reach backend' }, { status: 500 });
  }
}

export async function POST(request: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const { path } = await params;
  const safePath = sanitizePath(path);
  if (!safePath) {
    return NextResponse.json({ error: 'Invalid path' }, { status: 400 });
  }

  try {
    const body = await request.text();
    const response = await fetch(`${BACKEND_URL}/ai/${safePath}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body,
    });
    const data = await response.json();
    return NextResponse.json(data, { status: response.status });
  } catch {
    return NextResponse.json({ error: 'Failed to reach backend' }, { status: 500 });
  }
}
