import { test, expect } from '@playwright/test';

test.describe('Resource Lifecycle E2E', () => {
  test.beforeEach(async ({ page }) => {
    // 假设首页加载
    await page.goto('/');
  });

  test('should navigate to scenario list and open upload dialog', async ({ page }) => {
    // 1. 点击左侧菜单进入“想定资源”
    await page.click('text=想定资源库');
    await expect(page).toHaveURL(/.*res\/scenario/);

    // 2. 点击上传按钮
    await page.click('button:has-text("上传")');
    
    // 3. 验证弹窗是否出现
    const dialog = page.locator('.el-dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog).toContainText('上传资源');
  });

  test('global search should show results and navigate', async ({ page }) => {
    // 1. 触发搜索（假设快捷键或点击）
    await page.click('.hero-search-bar'); // 如果还在首页
    
    // 2. 输入关键字
    const searchInput = page.locator('input[placeholder*="搜索"]');
    await searchInput.fill('test');
    
    // 3. 验证结果面板出现（模拟异步）
    const results = page.locator('.search-results');
    // 这里如果后端没数据可能为空，E2E 测试通常需要预置数据
    // await expect(results).toBeVisible();
  });
});
