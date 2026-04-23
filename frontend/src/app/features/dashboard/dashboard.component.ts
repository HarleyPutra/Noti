import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';

// 1. EXACTLY matching the official GitHub reference
import { GridsterConfig, Gridster, GridsterItem, GridsterItemConfig } from 'angular-gridster2';

// Custom Widgets
import { TimerComponent } from './timer/timer.component';
import { EditorComponent } from './editor/editor.component';
import { StickerComponent } from './sticker/sticker.component';
import { ClockComponent } from './clock/clock.component';

// ─── INTERFACES ─────────────────────────────────────────────────────────

// 2. We extend GridsterItemConfig for our data, explicitly allowing 'type'
export interface DashboardWidget extends GridsterItemConfig {
  type?: string;
  id?: string;
}

export interface DashboardPreset {
  id: string;
  name: string;
  layout: DashboardWidget[];
}

// ─── COMPONENT ──────────────────────────────────────────────────────────

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    CommonModule,
    Gridster,      // 3. The exact standalone component from the docs
    GridsterItem,  // 4. The exact standalone item from the docs
    TimerComponent,
    EditorComponent,
    StickerComponent,
    ClockComponent
  ],
  templateUrl: './dashboard.component.html'
})
export class DashboardComponent implements OnInit {
  options: GridsterConfig = {};

  dashboard: DashboardWidget[] = [];
  presets: DashboardPreset[] = [];
  activePresetId: string = '';

  ngOnInit() {
    this.options = {
      draggable: { enabled: true },
      resizable: { enabled: true },
      pushItems: true,
      gridType: 'fit',
      minCols: 10,
      minRows: 10,
    };

    // Initialize layouts with explicit IDs for the tracker
    this.presets = [
      {
        id: 'preset-1',
        name: 'To-do',
        layout: [
          { id: 'w1', cols: 6, rows: 8, y: 0, x: 0, type: 'MAIN_EDITOR' },
          { id: 'w2', cols: 2, rows: 4, y: 0, x: 6, type: 'STICKER' },
          { id: 'w3', cols: 4, rows: 2, y: 8, x: 0, type: 'TIMER' },
          { id: 'w4', cols: 2, rows: 2, y: 4, x: 6, type: 'CLOCK' }
        ]
      },
      {
        id: 'preset-2',
        name: 'URGENT',
        layout: [
          { id: 'w5', cols: 8, rows: 10, y: 0, x: 0, type: 'MAIN_EDITOR' },
          { id: 'w6', cols: 2, rows: 2, y: 0, x: 8, type: 'CLOCK' }
        ]
      }
    ];

    this.selectPreset(this.presets[0]);
  }

  selectPreset(preset: DashboardPreset) {
    this.activePresetId = preset.id;
    this.dashboard = preset.layout;
  }

  addPreset() {
    const newPreset: DashboardPreset = {
      id: Date.now().toString(),
      name: `Page ${this.presets.length + 1}`,
      layout: []
    };

    this.presets.push(newPreset);
    this.selectPreset(newPreset);
  }
}
