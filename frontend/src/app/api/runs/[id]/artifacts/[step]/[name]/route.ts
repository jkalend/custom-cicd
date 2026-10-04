import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:8000';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ id: string; step: string; name: string }> }
) {
  try {
    const { id, step, name } = await params;
    const response = await fetch(
      `${BACKEND_URL}/runs/${encodeURIComponent(id)}/artifacts/${encodeURIComponent(step)}/${encodeURIComponent(name)}`,
      {
        method: 'GET',
      }
    );

    if (!response.ok) {
      return NextResponse.json(
        { success: false, error: 'Artifact not found' },
        { status: response.status }
      );
    }

    return new Response(response.body, {
      status: response.status,
      headers: {
        'Content-Type': response.headers.get('content-type') || 'application/octet-stream',
        'Content-Disposition': `inline; filename="${encodeURIComponent(name)}"`,
      },
    });
  } catch (error) {
    console.error('Failed to get artifact file:', error);
    return NextResponse.json(
      { success: false, error: error instanceof Error ? error.message : 'Failed to reach backend' },
      { status: 502 }
    );
  }
}
