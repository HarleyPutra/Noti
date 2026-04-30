import { Component, OnInit } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { AuthService } from './core/services/auth.service';
import { DbService } from './core/services/db.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet],
  templateUrl: './app.html'   // ← point to your existing file
})
export class AppComponent implements OnInit {
  constructor(private auth: AuthService, private db: DbService) {}

  async ngOnInit() {
    await this.db.init();
    await this.auth.init();
  }
}
