import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
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

  const response = NextResponse.next()

  // Only set the Cross-Origin-Resource-Policy header for specific domains
  if (isHostAllowed) {
    response.headers.set('Access-Control-Allow-Origin', `https://${host}`)
    response.headers.set('Access-Control-Allow-Credentials', 'true')
    
    // Set Cross-Origin-Resource-Policy based on the host
    if (host.includes('staging.hatchet-tools.com')) {
      response.headers.set('Cross-Origin-Resource-Policy', 'cross-origin')
    } else {
      response.headers.set('Cross-Origin-Resource-Policy', 'same-site')
    }
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
