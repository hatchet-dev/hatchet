import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
  const response = NextResponse.next()

  // Get the origin header from the request
  const origin = request.headers.get('origin')

  // Only set the header for specific domains
  const allowedDomains = ['https://docs.onhatchet.run']

  if (origin && allowedDomains.includes(origin)) {
    response.headers.set('Cross-Origin-Resource-Policy', 'same-site')
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
