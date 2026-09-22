import { createBrowserRouter, Outlet, redirect, RouterProvider } from "react-router"

import { Pages } from "./pages"

const router = createBrowserRouter([
    {
        element: <Outlet />,
        children: [
            {
                index: true,
                loader: () => redirect("/home"),
            },
            {
                path: "/home",
                element: <Pages.Home />,
            },
            {
                path: "/examples/new",
                element: <Pages.Example />,
            },
            {
                path: "*",
                element: <main>Page not found</main>,
            },
        ],
    },
])

export const AppRoutes = () => <RouterProvider router={router} />
