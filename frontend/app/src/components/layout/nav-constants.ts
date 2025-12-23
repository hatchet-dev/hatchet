// Shared constants for the v1 shell navigation (sidebar + header alignment).

export const V1_SIDEBAR_WIDTH_LEGACY_KEY = 'v1SidebarWidth';
export const V1_SIDEBAR_WIDTH_EXPANDED_KEY = 'v1SidebarWidthExpanded';
export const V1_SIDEBAR_COLLAPSED_KEY = 'v1SidebarCollapsed';

// Widths (px)
export const V1_DEFAULT_EXPANDED_SIDEBAR_WIDTH = 200; // matches prior `w-64`
export const V1_MIN_EXPANDED_SIDEBAR_WIDTH = 200;
export const V1_MAX_EXPANDED_SIDEBAR_WIDTH = 520;
export const V1_COLLAPSED_SIDEBAR_WIDTH = 56;

// Behavior
export const V1_COLLAPSE_SNAP_AT = V1_MIN_EXPANDED_SIDEBAR_WIDTH;
export const V1_EXPAND_SNAP_AT = V1_MIN_EXPANDED_SIDEBAR_WIDTH - 100;
export const V1_RESIZE_DRAG_THRESHOLD_PX = 3;
