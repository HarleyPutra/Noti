import { Component } from '@angular/core';
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
  loading = false;
  error = '';

  constructor(private auth: AuthService, private router: Router) {}

  async login() {
    this.loading = true;
    this.error = '';
    try {
      await this.auth.login();
      this.router.navigate(['/todos']);
    } catch (e: any) {
      this.error = 'Login failed. Please try again.';
    } finally {
      this.loading = false;
    }
  }
}
