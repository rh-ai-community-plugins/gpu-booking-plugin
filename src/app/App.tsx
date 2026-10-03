import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import CommunityBanner from './components/CommunityBanner';
import BookingPage from '~/components/BookingPage';
import AdminPage from '~/components/AdminPage';
import HelpPage from '~/components/HelpPage';

const App: React.FC = () => (
  <div className="community-plugin-layout">
    {/* [SHARED] Do not remove — all community plugins must display the CommunityBanner */}
    <CommunityBanner />
    <div className="community-plugin-content">
      <Routes>
        <Route path="/" element={<Navigate to="bookings" replace />} />
        <Route path="bookings/*" element={<BookingPage />} />
        <Route path="admin/*" element={<AdminPage />} />
        <Route path="help/:topic?" element={<HelpPage />} />
      </Routes>
    </div>
  </div>
);

export default App;
