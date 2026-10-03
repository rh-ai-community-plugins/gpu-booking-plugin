// [SHARED] Common section for all community plugins — never changes across plugins.
// Do not change the id or name: all community plugins share this section
// so they appear grouped together in the dashboard sidebar.
export const communityPluginsSectionExtension = {
  type: 'app.navigation/section' as const,
  properties: {
    id: 'community-plugins', // [SHARED] common section for all community plugins
    title: 'Community plugins', // [SHARED]
    group: '9_plugins', // [SHARED]
    iconRef: () => import(/* webpackMode: "eager" */ './CommunityNavIcon'),
  },
};

// [PLUGIN-SPECIFIC] Everything below is specific to this plugin

export const gpuBookingAreaExtension = {
  type: 'app.area' as const,
  properties: {
    id: 'gpu-booking', // [PLUGIN-SPECIFIC] unique area ID
    featureFlags: [] as string[],
  },
};

export const gpuBookingSectionExtension = {
  type: 'app.navigation/section' as const,
  properties: {
    id: 'gpu-booking', // [PLUGIN-SPECIFIC] unique nav section ID
    title: 'GPU Booking', // [PLUGIN-SPECIFIC] display name in sidebar
    group: '1_gpu_booking', // [PLUGIN-SPECIFIC] sort key within community-plugins
    section: 'community-plugins', // [SHARED] must match communityPluginsSectionExtension.id — do not change
    iconRef: () => import(/* webpackMode: "eager" */ '~/app/components/GpuBookingNavIcon'),
  },
};

export const bookingsNavExtension = {
  type: 'app.navigation/href' as const,
  properties: {
    id: 'gpu-booking-bookings',
    title: 'Bookings',
    href: '/gpu-booking/bookings',
    section: 'gpu-booking',
    path: '/gpu-booking/bookings/*',
  },
};

export const adminNavExtension = {
  type: 'app.navigation/href' as const,
  properties: {
    id: 'gpu-booking-admin',
    title: 'Admin',
    href: '/gpu-booking/admin',
    section: 'gpu-booking',
    path: '/gpu-booking/admin/*',
  },
};

export const helpNavExtension = {
  type: 'app.navigation/href' as const,
  properties: {
    id: 'gpu-booking-help',
    title: 'Help',
    href: '/gpu-booking/help',
    section: 'gpu-booking',
    path: '/gpu-booking/help/*',
  },
};

export const gpuBookingRouteExtension = {
  type: 'app.route' as const,
  properties: {
    path: '/gpu-booking/*', // [PLUGIN-SPECIFIC] top-level route prefix
    component: () => import(/* webpackMode: "eager" */ '~/app/App'),
  },
};

export const extensions = [
  communityPluginsSectionExtension,
  gpuBookingAreaExtension,
  gpuBookingSectionExtension,
  bookingsNavExtension,
  adminNavExtension,
  helpNavExtension,
  gpuBookingRouteExtension,
];

export default extensions;
