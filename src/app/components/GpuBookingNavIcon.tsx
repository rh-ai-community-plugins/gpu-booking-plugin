import React from 'react';

// [PLUGIN-SPECIFIC] Sidebar icon for the GPU Booking plugin — a GPU card glyph.
const GpuBookingIcon: React.FC = () => (
  <svg
    className="pf-v6-svg"
    viewBox="0 0 32 32"
    fill="currentColor"
    aria-hidden="true"
    role="img"
    width="1em"
    height="1em"
  >
    {/* Card body */}
    <rect
      x="2"
      y="8"
      width="28"
      height="16"
      rx="2"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
    />
    {/* GPU chip */}
    <rect x="8" y="12" width="8" height="8" />
    {/* Memory modules */}
    <rect x="19" y="12" width="3" height="8" />
    <rect x="24" y="12" width="3" height="8" />
    {/* Bracket pins */}
    <path
      d="M8 24v3M12 24v3M16 24v3M20 24v3M24 24v3"
      stroke="currentColor"
      strokeWidth="2"
      fill="none"
    />
  </svg>
);

export default GpuBookingIcon;
