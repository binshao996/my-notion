import { test, expect } from "@playwright/test";

test.describe("authentication", () => {
  test.beforeEach(async ({ page }) => {
    // Clear localStorage between tests
    await page.goto("/");
    await page.evaluate(() => localStorage.clear());
  });

  test("login with valid credentials", async ({ page }) => {
    await page.route("**/api/v1/auth/login", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          token: "test-jwt",
          user: {
            id: 1,
            email: "test@test.com",
            name: "Test User",
            avatar_url: "",
          },
        }),
      });
    });

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

    await page.goto("/login");

    // Verify login page is shown
    await expect(page.getByText("Sign in to My Notion")).toBeVisible();

    // Fill credentials
    await page.fill("#email", "test@test.com");
    await page.fill("#password", "password123");

    // Submit form
    await page.click('button[type="submit"]');

    // Verify redirect to workspace
    await expect(page).toHaveURL(/\/workspace\//);
    await expect(page.getByText("Welcome to My Notion")).toBeVisible();
  });

  test("login with invalid credentials shows error", async ({ page }) => {
    await page.route("**/api/v1/auth/login", async (route) => {
      await route.fulfill({
        status: 401,
        contentType: "application/json",
        body: JSON.stringify({ error: "Invalid email or password" }),
      });
    });

    await page.goto("/login");

    // Fill credentials
    await page.fill("#email", "wrong@test.com");
    await page.fill("#password", "wrongpass");

    // Submit form
    await page.click('button[type="submit"]');

    // Verify error message is shown
    const errorEl = page.locator(".bg-red-50");
    await expect(errorEl).toBeVisible();
    await expect(errorEl).toContainText("Invalid email or password");
  });

  test("register new user", async ({ page }) => {
    await page.route("**/api/v1/auth/register", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          token: "test-jwt-register",
          user: {
            id: 2,
            email: "newuser@test.com",
            name: "New User",
            avatar_url: "",
          },
        }),
      });
    });

    await page.route("**/api/v1/workspaces/1/tree", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify([]),
      });
    });

    await page.goto("/register");

    // Verify register page is shown
    await expect(page.getByText("Create your account")).toBeVisible();

    // Fill the form
    await page.fill("#name", "New User");
    await page.fill("#email", "newuser@test.com");
    await page.fill("#password", "password123");

    // Submit form
    await page.click('button[type="submit"]');

    // Verify redirect to workspace
    await expect(page).toHaveURL(/\/workspace\//);
  });

  test("logout redirects to login", async ({ page }) => {
    // Set token so ProtectedRoute lets us in
    await page.evaluate(() =>
      localStorage.setItem("token", "test-jwt")
    );

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

    await page.goto("/workspace/1");

    // Verify we're on the workspace page
    await expect(page.getByText("Sign out")).toBeVisible();

    // Click sign out
    await page.click("text=Sign out");

    // Verify redirect to login
    await expect(page).toHaveURL(/\/login/);
    await expect(page.getByText("Sign in to My Notion")).toBeVisible();
  });
});
