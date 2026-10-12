export const UNKNOWN_PAGE_COUNT = -1;

export const isPageCountKnown = (pageCount: number) =>
  pageCount !== UNKNOWN_PAGE_COUNT;
