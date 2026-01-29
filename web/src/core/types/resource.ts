export type ResourceState = 'ACTIVE' | 'READY' | 'PROCESSING' | 'PENDING' | 'FAILED' | string;
export type ResourceScope = 'PRIVATE' | 'PUBLIC';

export interface Category {
  id: string;
  name: string;
  parent_id?: string;
}

export interface CategoryNode extends Category {
  children?: CategoryNode[];
}

export interface ResourceVersion {
  id: string;
  resource_id: string;
  version_num: number;
  semver?: string;
  state: ResourceState;
  file_size: number;
  download_url?: string;
  meta_data?: Record<string, any>;
  created_at: string;
}

export interface Resource {
  id: string;
  name: string;
  type_key: string;
  category_id?: string;
  owner_id: string;
  scope: ResourceScope;
  tags: string[];
  created_at: string;
  updated_at: string;
  latest_version_id?: string;
  latest_version?: ResourceVersion;
}

export interface ResourceDependency {
  id: string;
  resource_id: string;
  resource_name: string;
  type_key?: string;
  version_id: string;
  semver?: string;
  constraint?: string;
  dependencies?: ResourceDependency[];
}
