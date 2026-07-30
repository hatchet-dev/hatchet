import React from "react";

type Tag = "production" | "staging";
type Role = "ADMIN" | "MEMBER";

const TAG_STYLES: Record<Tag, string> = {
  production: "bg-[#2a78d6] dark:bg-[#3987e5]",
  staging: "bg-[#eb6834] dark:bg-[#d95926]",
};

function TagChip({ tag }: { tag: Tag }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-white ${TAG_STYLES[tag]}`}
    >
      {tag}
    </span>
  );
}

function RoleBadge({ role }: { role: Role }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center rounded border border-black/15 dark:border-white/20 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-[#52514e] dark:text-[#c3c2b7] ${
        role === "ADMIN" ? "font-semibold" : "font-medium"
      }`}
    >
      {role}
    </span>
  );
}

function UserRow({ email, role }: { email: string; role: Role }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="truncate font-mono text-xs text-[#52514e] dark:text-[#c3c2b7]">
        {email}
      </span>
      <RoleBadge role={role} />
    </div>
  );
}

type BoxProps = {
  kind: "USER GROUP" | "TENANT";
  title: string;
  role?: Role;
  tags: Tag[];
  x: number;
  y: number;
  width: number;
  height: number;
  children: React.ReactNode;
};

function DiagramBox({
  kind,
  title,
  role,
  tags,
  x,
  y,
  width,
  height,
  children,
}: BoxProps) {
  return (
    <foreignObject x={x} y={y} width={width} height={height}>
      <div
        className={`flex h-full w-full flex-col gap-1.5 rounded-lg border bg-[#fcfcfb] dark:bg-[#1a1a19] border-black/10 dark:border-white/10 p-3 shadow-sm box-border ${
          kind === "USER GROUP" ? "border-dashed" : "border-solid"
        }`}
      >
        <span className="text-[10px] font-medium uppercase tracking-wide text-[#898781]">
          {kind}
        </span>
        <div className="flex items-center justify-between gap-2">
          <span className="text-sm font-semibold text-[#0b0b0b] dark:text-white">
            {title}
          </span>
          {role && <RoleBadge role={role} />}
        </div>
        <div className="flex flex-wrap gap-1">
          {tags.map((tag) => (
            <TagChip key={tag} tag={tag} />
          ))}
        </div>
        <hr className="my-0.5 border-t border-black/10 dark:border-white/10" />
        {children}
      </div>
    </foreignObject>
  );
}

const ROW = { 1: 100, 2: 330, 3: 560 } as const;
type RowNum = keyof typeof ROW;
const GROUP_X = 20;
const GROUP_RIGHT = 280;
const TENANT_X = 600;
const BOX_W = 260;
const BOX_H = 190;
const ARROW_COLOR = "#898781";
const VIEW_W = 880;
const VIEW_H = 660;

function boxTop(rowCenter: number) {
  return rowCenter - BOX_H / 2;
}

function isSubset(subset: Tag[], superset: Tag[]) {
  return subset.every((tag) => superset.includes(tag));
}

type Group = {
  id: string;
  title: string;
  tags: Tag[];
  role: Role;
  member: string;
  row: RowNum;
};

type Tenant = {
  id: string;
  title: string;
  tags: Tag[];
  row: RowNum;
};

const GROUPS: Group[] = [
  {
    id: "everyone",
    title: "Everyone",
    tags: ["production", "staging"],
    role: "ADMIN",
    member: "a@example.com",
    row: 1,
  },
  {
    id: "production-team",
    title: "Production Team",
    tags: ["production"],
    role: "MEMBER",
    member: "c@example.com",
    row: 2,
  },
  {
    id: "staging-team",
    title: "Staging Team",
    tags: ["staging"],
    role: "MEMBER",
    member: "b@example.com",
    row: 3,
  },
];

const TENANTS: Tenant[] = [
  { id: "preview", title: "Preview", tags: ["production", "staging"], row: 1 },
  { id: "production", title: "Production", tags: ["production"], row: 2 },
  { id: "staging", title: "Staging", tags: ["staging"], row: 3 },
];

// A tenant automatically syncs a group's member when the tenant's tags are a
// subset of the group's tags.
function syncedUsers(tenant: Tenant) {
  return GROUPS.filter((group) => isSubset(tenant.tags, group.tags)).map(
    (group) => ({
      email: group.member,
      role: group.role,
    }),
  );
}

const EDGES = GROUPS.flatMap((group) =>
  TENANTS.filter((tenant) => isSubset(tenant.tags, group.tags)).map(
    (tenant) => ({
      from: group.row,
      to: tenant.row,
    }),
  ),
);

export function UserGroupSyncDiagram() {
  return (
    <div className="not-prose my-8">
      <svg
        viewBox={`0 0 ${VIEW_W} ${VIEW_H}`}
        className="block w-full h-auto"
        role="img"
        aria-label="Diagram showing three user groups syncing their members and roles into three tenants based on tag matching"
      >
        <defs>
          <marker
            id="ug-sync-arrow"
            viewBox="0 0 10 10"
            refX="8"
            refY="5"
            markerWidth={7}
            markerHeight={7}
            orient="auto-start-reverse"
          >
            <path d="M0,0 L10,5 L0,10 z" fill={ARROW_COLOR} />
          </marker>
        </defs>

        {EDGES.map(({ from, to }) =>
          from === to ? (
            <line
              key={`${from}-${to}`}
              x1={GROUP_RIGHT}
              y1={ROW[from]}
              x2={TENANT_X - 2}
              y2={ROW[to]}
              stroke={ARROW_COLOR}
              strokeWidth={2}
              markerEnd="url(#ug-sync-arrow)"
            />
          ) : (
            <path
              key={`${from}-${to}`}
              d={`M${GROUP_RIGHT},${ROW[from]} Q440,${ROW[from]} ${TENANT_X - 2},${ROW[to]}`}
              stroke={ARROW_COLOR}
              strokeWidth={2}
              fill="none"
              markerEnd="url(#ug-sync-arrow)"
            />
          ),
        )}

        {GROUPS.map((group) => (
          <DiagramBox
            key={group.id}
            kind="USER GROUP"
            title={group.title}
            role={group.role}
            tags={group.tags}
            x={GROUP_X}
            y={boxTop(ROW[group.row])}
            width={BOX_W}
            height={BOX_H}
          >
            <span className="text-[10px] uppercase tracking-wide text-[#898781]">
              Member
            </span>
            <span className="font-mono text-xs text-[#52514e] dark:text-[#c3c2b7]">
              {group.member}
            </span>
          </DiagramBox>
        ))}

        {TENANTS.map((tenant) => (
          <DiagramBox
            key={tenant.id}
            kind="TENANT"
            title={tenant.title}
            tags={tenant.tags}
            x={TENANT_X}
            y={boxTop(ROW[tenant.row])}
            width={BOX_W}
            height={BOX_H}
          >
            <span className="text-[10px] uppercase tracking-wide text-[#898781]">
              Synced users
            </span>
            {syncedUsers(tenant).map((user) => (
              <UserRow key={user.email} email={user.email} role={user.role} />
            ))}
          </DiagramBox>
        ))}
      </svg>
      <p className="mt-3 text-center text-xs text-[#898781]">
        Dashed boxes are user groups, solid boxes are tenants. An arrow means
        the tenant's tags are a subset of the group's tags, so the group's
        member is added automatically with the group's role.
      </p>
    </div>
  );
}
