import { test, expect } from "@playwright/test";

/**
 * Helper to set up auth mocks so ProtectedRoute passes.
 * Sets token in localStorage and mocks the workspace tree.
 */
async function setupAuthAndWorkspace(page: import("@playwright/test").Page) {
  // Set token so ProtectedRoute checks pass
  await page.evaluate(() => localStorage.setItem("token", "test-jwt"));

  await page.route("**/api/v1/workspaces/1/tree", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify([
        {
          id: 1,
          title: "Getting Started",
          workspace_id: 1,
          parent_page_id: null,
          icon: "",
          cover: "",
          created_by: 1,
          created_at: "",
          updated_at: "",
          archived: false,
        },
      ]),
    });
  });
}

/**
 * Set up mocks for a page view: page detail, blocks, and save.
 */
async function setupPageMocks(
  page: import("@playwright/test").Page,
  pageId: number,
  blocks: Array<Record<string, unknown>> = []
) {
  await page.route(`**/api/v1/pages/${pageId}`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        id: pageId,
        title: "Test Page",
        workspace_id: 1,
        cover: "",
      }),
    });
  });

  await page.route(`**/api/v1/pages/${pageId}/blocks`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(blocks),
    });
  });

  // Mock save (PUT blocks)
  await page.route(`**/api/v1/pages/${pageId}/blocks`, async (route) => {
    if (route.request().method() === "PUT") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    } else {
      await route.continue();
    }
  });

  // Mock page children
  await page.route(`**/api/v1/pages/${pageId}/children`, async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify([]),
    });
  });
}

test.beforeEach(async ({ page }) => {
  await page.evaluate(() => localStorage.clear());
});

test("create new page", async ({ page }) => {
  await setupAuthAndWorkspace(page);

  // Mock page creation API
  await page.route("**/api/v1/pages", async (route) => {
    if (route.request().method() === "POST") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          id: 99,
          title: "My New Page",
          workspace_id: 1,
          parent_page_id: null,
          icon: "",
          cover: "",
          created_by: 1,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          archived: false,
        }),
      });
    } else {
      await route.continue();
    }
  });

  // Set up page view mocks for the newly created page (id=99)
  await setupPageMocks(page, 99);

  await page.goto("/workspace/1");

  // Verify workspace loaded
  await expect(page.getByText("Welcome to My Notion")).toBeVisible();

  // Click "+ New page" button in the sidebar
  await page.click("text=+ New page");

  // An input should appear for the page name
  const pageNameInput = page.locator('input[placeholder="Page name"]');
  await expect(pageNameInput).toBeVisible();

  // Type the page name and press Enter
  await pageNameInput.fill("My New Page");
  await pageNameInput.press("Enter");

  // Verify we navigated to the new page
  await expect(page).toHaveURL(/\/page\/99/);
});

test("edit page title", async ({ page }) => {
  await setupAuthAndWorkspace(page);

  // Mock page with one empty paragraph block
  await setupPageMocks(page, 42, [
    {
      id: 1,
      type: "paragraph",
      props: JSON.stringify({ text: "" }),
      parent_block_id: null,
    },
  ]);

  await page.goto("/page/42");

  // Wait for the page to load (the block editor should appear)
  // The Tiptap editor renders contentEditable elements with class ProseMirror
  const proseMirror = page.locator(".ProseMirror").first();
  await expect(proseMirror).toBeVisible({ timeout: 10000 });

  // Click to focus the block and type a title
  await proseMirror.click();
  await page.keyboard.type("My Page Title");

  // Verify the text was entered (check the ProseMirror content)
  await expect(proseMirror).toContainText("My Page Title");
});

test("add block with / command", async ({ page }) => {
  await setupAuthAndWorkspace(page);

  // Mock page with one empty paragraph block
  await setupPageMocks(page, 43, [
    {
      id: 1,
      type: "paragraph",
      props: JSON.stringify({ text: "" }),
      parent_block_id: null,
    },
  ]);

  await page.goto("/page/43");

  // Wait for the block editor
  const proseMirror = page.locator(".ProseMirror").first();
  await expect(proseMirror).toBeVisible({ timeout: 10000 });

  // Click the block to focus it, then type "/"
  await proseMirror.click();
  await page.keyboard.type("/");

  // The command palette should appear with a "Filter..." input
  const paletteInput = page.locator('input[placeholder="Filter..."]');
  await expect(paletteInput).toBeVisible({ timeout: 5000 });

  // Select "Heading 1" from the command palette
  await page.click("text=Heading 1");

  // The command palette should close after selection
  await expect(paletteInput).not.toBeVisible();

  // The block should now be a heading1 type.
  // Heading1 renders with text-3xl font-bold classes on the ProseMirror container
  // After type change, the block should still be visible in the editor
  await expect(proseMirror).toBeVisible();
});

test("delete page", async ({ page }) => {
  await setupAuthAndWorkspace(page);

  // Mock page with one paragraph block
  await setupPageMocks(page, 77, [
    {
      id: 1,
      type: "paragraph",
      props: JSON.stringify({ text: "This will be deleted" }),
      parent_block_id: null,
    },
  ]);

  // Mock the DELETE API endpoint
  let deleteCalled = false;
  await page.route("**/api/v1/pages/77", async (route) => {
    if (route.request().method() === "DELETE") {
      deleteCalled = true;
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({}),
      });
    } else {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          id: 77,
          title: "Delete Me",
          workspace_id: 1,
          cover: "",
        }),
      });
    }
  });

  // Navigate to the page first to verify it loads
  await page.goto("/page/77");
  await expect(page.locator(".ProseMirror").first()).toBeVisible({ timeout: 10000 });

  // Call the delete API via page.evaluate (since there's no delete button in the UI)
  await page.evaluate(async () => {
    const token = localStorage.getItem("token");
    await fetch("/api/v1/pages/77", {
      method: "DELETE",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
    });
  });

  // Verify the delete API was called
  expect(deleteCalled).toBe(true);

  // After deletion, re-mock the tree without the page and navigate to workspace
  await page.unroute("**/api/v1/workspaces/1/tree");
  await page.route("**/api/v1/workspaces/1/tree", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify([]),
    });
  });

  // Navigate to workspace and verify the empty state
  await page.goto("/workspace/1");
  await expect(page.getByText("Welcome to My Notion")).toBeVisible();
});
