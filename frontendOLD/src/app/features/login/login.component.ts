import { Component, inject, effect } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router } from '@angular/router';
import { AuthService } from '../../core/services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './login.component.html'
})
export class LoginComponent {
  private auth = inject(AuthService);
  private router = inject(Router);

  loading = false;
  error = '';

  constructor() {
    effect(() => {
      if (this.auth.currentUser()) {
        this.router.navigate(['/dashboard']);
      }
    });
  }

  async login() {
    try {
      this.loading = true;
      this.error = '';
      
      await this.auth.login();
      
    } catch (err: any) {
      this.error = err.message || 'Failed to login';
    } finally {
      this.loading = false;
    }
  }
}