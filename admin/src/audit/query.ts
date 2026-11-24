export const pageSize = 50;

export const filterFields = ["actor", "object_type", "object_id", "action", "reason", "from", "to"] as const;

export type FilterField = (typeof filterFields)[number];

export type SearchParams = Readonly<Record<string, string | string[] | undefined>>;

export type AuditFilter = Readonly<Partial<Record<FilterField, string>>>;

export type AuditEntry = {
  id: number;
  at: string;
  actor_id: string;
  actor_roles: string[];
  environment: string;
  action: string;
  object_type: string;
  object_id: string;
  before?: unknown;
  after?: unknown;
  reason?: string;
  approval_id?: number;
  ref?: string;
  change_set?: string;
  config_version?: number;
  session_id: string;
  source_ip?: string;
  user_agent?: string;
};

const minuteInput = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/;

function single(value: string | string[] | undefined): string {
  return (Array.isArray(value) ? value[0] : value)?.trim() ?? "";
}

export function readFilter(params: SearchParams): AuditFilter {
  const filter: Partial<Record<FilterField, string>> = {};
  for (const field of filterFields) {
    const value = single(params[field]);
    if (value) {
      filter[field] = value;
    }
  }
  return filter;
}

export function utcInstant(minute: string): string {
  return minuteInput.test(minute) ? `${minute}:00Z` : minute;
}

function filterQuery(filter: AuditFilter): URLSearchParams {
  const query = new URLSearchParams();
  for (const field of filterFields) {
    const value = filter[field];
    if (value) {
      query.set(field, field === "from" || field === "to" ? utcInstant(value) : value);
    }
  }
  return query;
}

export function auditExportPath(filter: AuditFilter): string {
  return `/admin/audit/export?${filterQuery(filter)}`;
}

export function auditApiPath(filter: AuditFilter, beforeId: string): string {
  const query = filterQuery(filter);
  if (beforeId) {
    query.set("before_id", beforeId);
  }
  query.set("limit", String(pageSize));
  return `/admin/audit?${query}`;
}

export function olderPageHref(filter: AuditFilter, entries: readonly AuditEntry[]): string | null {
  if (entries.length < pageSize) {
    return null;
  }
  const query = new URLSearchParams(filter as Record<string, string>);
  query.set("before_id", String(entries[entries.length - 1].id));
  return `/audit?${query}`;
}

export function beforeIdOf(params: SearchParams): string {
  const value = single(params.before_id);
  return /^[1-9]\d*$/.test(value) ? value : "";
}
