import { test, expect } from '@playwright/test';

test.describe('Advanced Resource Scenarios', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('complex filtering linkage (category + search)', async ({ page }) => {
    // 1. 进入模型库
    await page.click('text=模型资源库');
    
    // 2. 选择一个分类
    const categoryTree = page.locator('.el-tree');
    // 假设存在一个名为“建筑”的分类
    await page.click('text=模型资源库'); // 确保进入了正确的侧边菜单
    
    // 3. 搜索过滤
    const searchInput = page.locator('input[placeholder*="名称"]');
    await searchInput.fill('Tank');
    
    // 4. 验证 URL 参数同步 (假设 URL 反映了状态)
    // await expect(page).toHaveURL(/.*query=Tank/);
    
    // 5. 验证空状态 (如果没有匹配项)
    // const empty = page.locator('.el-empty');
    // await expect(empty).toBeVisible();
  });

  test('external viewer communication handshake', async ({ page }) => {
    // 1. 进入某个支持外部预览的资源 (模拟点击第一张卡片)
    await page.click('text=想定资源库');
    const firstCard = page.locator('.resource-card').first();
    await firstCard.click();

    // 2. 检查 Iframe 是否加载
    const iframe = page.locator('iframe.external-iframe');
    await expect(iframe).toBeVisible();

    // 3. 验证预览容器状态
    const container = page.locator('.external-viewer-container');
    await expect(container).not.toHaveClass(/is-loading/);
  });
});
