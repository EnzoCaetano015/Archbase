import { createBrowserRouter, Navigate, Outlet, RouterProvider } from "react-router"

import { ExamplePage } from "../pages/Example/Example"

const RootLayout = () => <Outlet />

const router = createBrowserRouter([
    {
        element: <RootLayout />,
        children: [
            {
                index: true,
                element: <Navigate to="/examples" replace />,
            },
            {
                path: "/examples",
                element: <ExamplePage />,
            },
            {
                path: "*",
                element: <main>Page not found</main>,
            },
        ],
    },
])

export const AppRoutes = () => <RouterProvider router={router} />
