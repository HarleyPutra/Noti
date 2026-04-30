import { CapacitorConfig } from '@capacitor/cli';

const config: CapacitorConfig = {
  appId: 'com.bunni.noti',
  appName: 'Noti',
  webDir: 'dist/frontend/browser',
  plugins: {
    GoogleAuth: {
      scopes: ['profile', 'email',
               'https://www.googleapis.com/auth/drive.appdata'],
      serverClientId: '572458489581-pcpgp989n9k2v6vcobq0jhe7c8jr5uid.apps.googleusercontent.com',
      forceCodeForRefreshToken: true
    }
  }
};
export default config;
