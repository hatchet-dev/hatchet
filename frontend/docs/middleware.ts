import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  // Get the host header (the domain being requested)
  const host = request.headers.get('host')
  const origin = request.headers.get('origin')


  const allowedDomains = ['staging.hatchet-tools.com', '*.onhatchet.run', '*.hatchet.run']

  const isOriginAllowed = origin && allowedDomains.some(domain => {
    if (domain.startsWith('*.')) {
      const suffix = domain.slice(2)
      return origin.endsWith(suffix)
    }
    return origin.includes(domain)
  })

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

  const response = NextResponse.next()

  if (isOriginAllowed) {
    response.headers.set('Access-Control-Allow-Origin', origin)
    response.headers.set('Access-Control-Allow-Credentials', 'true')
    response.headers.set('Cross-Origin-Resource-Policy', 'cross-origin')
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
