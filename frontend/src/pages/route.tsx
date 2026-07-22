import { createHashRouter, Navigate, RouterProvider } from 'react-router'
import Layout from './Layout'
import CleanupPage from './cleanup'
import CleanupDetailPage from './cleanup/[id]'
import HomePage from './home'
import MigrationPage from './migration'
import SettingsPage from './settings'

const router = createHashRouter([
  {
    path: '/',
    Component: Layout,
    children: [
      {
        index: true,
        element: <Navigate to="/home" replace />,
      },
      {
        path: 'home',
        Component: HomePage,
      },
      {
        path: 'cleanup',
        Component: CleanupPage,
      },
      {
        path: 'cleanup/:id',
        Component: CleanupDetailPage,
      },
      {
        path: 'migration',
        Component: MigrationPage,
      },
      {
        path: 'settings',
        Component: SettingsPage,
      },
    ],
  },
])

export default function AppRoutes() {
  return <RouterProvider router={router} />
}
