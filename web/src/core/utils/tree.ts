/**
 * 通用树形结构构建工具
 */
export interface TreeNode {
  id: string;
  parent_id?: string;
  children?: TreeNode[];
  [key: string]: any;
}

/**
 * 将扁平列表转换为树形结构
 * @param list 原始扁平列表
 * @param parentId 根节点父 ID，默认为空字符或 undefined
 */
export function buildTree<T extends TreeNode>(list: T[], parentId: string | null | undefined = ""): T[] {
  return list
    .filter(item => {
      // 处理多种空值情况
      if (!parentId) return !item.parent_id;
      return item.parent_id === parentId;
    })
    .map(item => ({
      ...item,
      children: buildTree(list, item.id)
    }));
}
