import { Component, OnInit, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-clock',
  standalone: true,
  // CommonModule is required here so we can use the 'date' pipe in our HTML!
  imports: [CommonModule],
  templateUrl: './clock.component.html'
})
export class ClockComponent implements OnInit, OnDestroy {
  currentTime: Date = new Date();
  private timerId: any;

  ngOnInit() {
    // Update the time every 1000 milliseconds (1 second)
    this.timerId = setInterval(() => {
      this.currentTime = new Date();
    }, 1000);
  }

  // CRITICAL: Stop the clock if the widget is deleted or the user leaves the page
  ngOnDestroy() {
    if (this.timerId) {
      clearInterval(this.timerId);
    }
  }
}
