import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

function markdownPath(pathname: string): string | null {
  if (
    pathname.startsWith('/api/') ||
    pathname.startsWith('/llms') ||
    pathname === '/' ||
    /\.[a-z0-9]+$/i.test(pathname)
  ) {
    return null
  }
  return `/llms${pathname}.md`
}

export default function proxy(request: NextRequest) {
  // Get the host header (the domain being requested)
  const host = request.headers.get('host')

  const allowedDomains = ['staging.hatchet-tools.com', '*.onhatchet.run', '*.hatchet.run']

  // Check if host is allowed for CORS
  const isHostAllowed = host && allowedDomains.some(domain => {
    if (domain.startsWith('*.')) {
      const suffix = domain.slice(2) // Remove *. prefix
      return host.endsWith(suffix)
    }
    return domain === host
  })

  // Handle preflight requests
  if (request.method === 'OPTIONS') {
    const response = new NextResponse(null, { status: 200 })

    if (isHostAllowed) {
      response.headers.set('Access-Control-Allow-Origin', `https://${host}`)
      response.headers.set('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
      response.headers.set('Access-Control-Allow-Headers', 'Content-Type, Authorization')
      response.headers.set('Access-Control-Max-Age', '86400')
    }

    return response
  }

  const { pathname } = request.nextUrl

  // Serve markdown to agents: /v1/tasks.md mirrors /llms/v1/tasks.md
  if (/\.md$/.test(pathname) && !pathname.startsWith('/llms')) {
    return NextResponse.rewrite(new URL(`/llms${pathname}`, request.url))
  }

  const mdPath = markdownPath(pathname)

  // Content negotiation: Accept: text/markdown gets the markdown mirror
  if (mdPath && request.headers.get('accept')?.includes('text/markdown')) {
    return NextResponse.rewrite(new URL(mdPath, request.url))
  }

  const response = NextResponse.next()

  response.headers.set('Access-Control-Allow-Origin', "*")
  response.headers.set('Access-Control-Allow-Credentials', 'true')
  response.headers.set('Cross-Origin-Resource-Policy', 'cross-origin')
  response.headers.set('Cross-Origin-Embedder-Policy', 'credentialless')

  if (mdPath) {
    response.headers.set('Link', `<${mdPath}>; rel="alternate"; type="text/markdown"`)
  }

  return response
}

// Configure which paths this middleware should run on
export const config = {
  matcher: [
    // Match all paths except static files and Next.js internals
    '/((?!_next/static|_next/image|favicon.ico).*)',
  ],
}
