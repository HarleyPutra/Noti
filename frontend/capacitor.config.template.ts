import { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.bunni.noti',
  appName: 'Noti',
  webDir: 'dist/frontend/browser',
  plugins: {
    GoogleAuth: {
      scopes: ['profile', 'email',
               'https://www.googleapis.com/auth/drive.appdata'],
      serverClientId: '',
      forceCodeForRefreshToken: true
    }
  }
};
export default config;
