import { Component, ViewEncapsulation } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { QuillModule } from 'ngx-quill';

@Component({
  selector: 'app-editor',
  standalone: true,
  // We import QuillModule and FormsModule to bind the text data
  imports: [CommonModule, FormsModule, QuillModule],
  templateUrl: './editor.component.html',
  styleUrls: ['./editor.component.css'],
  // Encapsulation.None allows our custom CSS to style the Quill inner elements
  encapsulation: ViewEncapsulation.None
})
export class EditorComponent {
  // This variable will hold all your rich text HTML!
  noteContent: string = '';

  // We configure the toolbar to have exactly what Notion has
  editorConfig = {
    toolbar: [
      ['bold', 'italic', 'underline', 'strike'],
      [{ 'list': 'bullet' }, { 'list': 'check' }], // Bullet points and Checklists!
      [{ 'header': [1, 2, 3, false] }],            // H1, H2, H3
      ['clean']                                    // Remove formatting
    ]
  };
}
