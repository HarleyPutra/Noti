import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-sticker',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './sticker.component.html'
})
export class StickerComponent {
  // These variables hold the current state of the widget
  imageUrl: string | null = null;
  emoji: string | null = null;

  // We start in 'edit' mode to show the upload buttons
  isEditing: boolean = true;

  // Triggered when the user selects a file from their computer
  onFileSelected(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0];
    if (file) {
      const reader = new FileReader();

      // Once the file is read, we save the Base64 string to our imageUrl
      reader.onload = () => {
        this.imageUrl = reader.result as string;
        this.emoji = null; // Clear out any existing emoji
        this.isEditing = false; // Hide the buttons
      };

      reader.readAsDataURL(file);
    }
  }

  // A helper to quickly set an emoji instead of an image
  setEmoji(selectedEmoji: string) {
    this.emoji = selectedEmoji;
    this.imageUrl = null;
    this.isEditing = false;
  }

  // Allows the user to reset the widget and pick something else
  clearSticker() {
    this.imageUrl = null;
    this.emoji = null;
    this.isEditing = true;
  }
}
