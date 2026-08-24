import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import {
  CONSENT_COOKIE,
  REGION_COOKIE,
  REGION_COOKIE_MAX_AGE,
  defaultStatusForRegion,
  resolveRegion,
  sharedCookieDomain,
} from '@/lib/consent'
import {
  ATTRIBUTION_COOKIE,
  ATTRIBUTION_COOKIE_MAX_AGE,
  nextAttributionCookie,
} from '@/lib/attribution'

/**
 * Publish the visitor's country so the consent layer can pick a regional
 * default without a client round trip, and so cloud.hatchet.run — which has no
 * edge geo header of its own — inherits the same answer via `.hatchet.run`.
 *
 * Vercel sets `x-vercel-ip-country`; when it is absent (local dev, other
 * hosts) we set nothing and the consent layer falls back to "restricted".
 */
function resolveCountry(request: NextRequest): string | null {
  return (
    process.env.CONSENT_REGION_OVERRIDE ||
    request.headers.get('x-vercel-ip-country') ||
    null
  )
}

function setRegionCookie(
  request: NextRequest,
  response: NextResponse,
  country: string | null,
) {
  if (!country) return

  response.cookies.set({
    name: REGION_COOKIE,
    value: country,
    path: '/',
    maxAge: REGION_COOKIE_MAX_AGE,
    sameSite: 'lax',
    domain: sharedCookieDomain(request.nextUrl.hostname),
    secure: request.nextUrl.protocol === 'https:',
  })
}

/**
 * Stash the ad click / campaign this visit arrived with, so a signup on
 * cloud.hatchet.run days later can still be attributed to it.
 *
 * This is a marketing cookie with a 90-day life, so it follows consent: a
 * visitor in the EEA, the UK or Switzerland gets nothing until they accept,
 * and an explicit decline clears whatever is already set.
 */
function setAttributionCookie(
  request: NextRequest,
  response: NextResponse,
  country: string | null,
) {
  const stored = request.cookies.get(CONSENT_COOKIE)?.value
  const consent =
    stored === 'granted' || stored === 'denied'
      ? stored
      : defaultStatusForRegion(resolveRegion(country))

  if (consent === 'denied') {
    if (request.cookies.has(ATTRIBUTION_COOKIE)) {
      response.cookies.set({
        name: ATTRIBUTION_COOKIE,
        value: '',
        path: '/',
        maxAge: 0,
        domain: sharedCookieDomain(request.nextUrl.hostname),
      })
    }
    return
  }

  const value = nextAttributionCookie(
    request.cookies.get(ATTRIBUTION_COOKIE)?.value,
    request.nextUrl.searchParams,
    request.nextUrl.pathname,
    request.headers.get('referer'),
    request.nextUrl.hostname,
  )
  if (!value) return

  response.cookies.set({
    name: ATTRIBUTION_COOKIE,
    value,
    path: '/',
    maxAge: ATTRIBUTION_COOKIE_MAX_AGE,
    sameSite: 'lax',
    domain: sharedCookieDomain(request.nextUrl.hostname),
    secure: request.nextUrl.protocol === 'https:',
  })
}

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

  const country = resolveCountry(request)
  setRegionCookie(request, response, country)
  setAttributionCookie(request, response, country)

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
