import { Component, OnDestroy } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms'; // <-- ADDED THIS IMPORT!

@Component({
  selector: 'app-timer',
  standalone: true,
  imports: [CommonModule, FormsModule], // <-- FIXED THE COMMA HERE!
  templateUrl: './timer.component.html'
})
export class TimerComponent implements OnDestroy {
  // 25 minutes default (in seconds)
  private readonly DEFAULT_TIME = 25 * 60;

  timeLeft: number = this.DEFAULT_TIME;
  isRunning: boolean = false;
  private timerInterval: any;

  isEditingTime: boolean = false;
  inputMinutes: number = 25;

  // This automatically calculates the MM:SS format whenever timeLeft changes
  get formattedTime(): string {
    const minutes = Math.floor(this.timeLeft / 60);
    const seconds = this.timeLeft % 60;
    return `${this.padZero(minutes)}:${this.padZero(seconds)}`;
  }

  // Helper to add a leading zero (e.g., '9' becomes '09')
  private padZero(num: number): string {
    return num < 10 ? '0' + num : num.toString();
  }

  saveCustomTime() {
    if (this.inputMinutes > 0) {
      this.timeLeft = this.inputMinutes * 60;
    }
    this.isEditingTime = false;
  }

  toggleTimer() {
    if (this.isRunning) {
      this.pauseTimer();
    } else {
      this.startTimer();
    }
  }

  private startTimer() {
    if (this.timeLeft <= 0) return; // Don't start if time is up

    this.isRunning = true;
    this.timerInterval = setInterval(() => {
      if (this.timeLeft > 0) {
        this.timeLeft--;
      } else {
        this.pauseTimer();
        // TODO: Trigger Wails Go backend later to play a system notification!
      }
    }, 1000); // Ticks every 1000 milliseconds (1 second)
  }

  private pauseTimer() {
    this.isRunning = false;
    if (this.timerInterval) {
      clearInterval(this.timerInterval);
    }
  }

  resetTimer() {
    this.pauseTimer();
    this.timeLeft = this.DEFAULT_TIME; // Note: You might want to change this to this.inputMinutes * 60 if you want it to reset to their custom time!
  }

  // CRITICAL: Stop the interval if the widget is destroyed/removed
  ngOnDestroy() {
    this.pauseTimer();
  }
}