import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: 'login',
    loadComponent: () =>
      import('./login/login.component').then(m => m.LoginComponent)
  },
  {
    path: '',
    loadComponent: () =>
      import('./note/note.component').then(m => m.NoteComponent)
  },
  {
    path: 'menu',
    loadComponent: () =>
      import('./menu/menu.component').then(m => m.MenuComponent)
  }
];
