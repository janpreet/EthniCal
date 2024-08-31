# EthniCal

EthniCal is an AI-powered global ethnic calendar generator that creates ICS files for various cultural and ethnic events worldwide. It uses AI to gather event information and generates an interactive web calendar for easy viewing and integration.

## Overview

This project aims to create a comprehensive calendar of ethnic and cultural events from around the world. It uses AI to gather event information, processes this data, and generates ICS files and an interactive web calendar. The calendar is designed to be easily integrated into personal calendar applications and viewed online.

## Features

- AI-powered event gathering for multiple ethnic and cultural groups
- Generation of ICS files for easy calendar integration
- Interactive web calendar interface
- Support for both single-day events and date ranges (e.g., multi-day festivals)
- Configurable event groups and categories

## How It Works

1. **AI Event Generation**: The system queries an AI model (currently using Claude) to gather event information for various ethnic and cultural groups.

2. **Event Parsing**: The AI responses are parsed to extract event names, dates, and date ranges.

3. **ICS File Generation**: The parsed events are used to create ICS files for each group and a combined file for all events.

4. **Web Calendar Creation**: An interactive web calendar is generated using FullCalendar, displaying all events with tooltips for additional information.

5. **GitHub Pages Deployment**: The generated calendar and ICS files are automatically deployed to GitHub Pages for easy access.

## Project Structure

- `main.go`: Main Go file containing the logic for AI querying, event parsing, and ICS file generation.
- `docs/`: Directory containing the web calendar files and generated ICS files.
- `configs/`: Directory containing JSON configuration files for different ethnic/cultural groups.
- `calendar_template.html`: HTML template for the interactive web calendar.

## Contributing

Contributions to EthniCal are welcome! Here are some ways you can contribute:

1. **Adding New Ethnic/Cultural Groups**: Create new JSON config files in the `configs/` directory for additional groups.

2. **Improving AI Prompts**: Enhance the AI querying process for more accurate and comprehensive event information.

3. **Fixing ICS Generation**: Improvements to the ICS file generation process are particularly welcome. If you notice any issues with the current ICS files or have ideas for enhancements, please submit a PR.

4. **Enhancing the Web Calendar**: Improve the design, functionality, or accessibility of the web calendar interface.

5. **Documentation**: Help improve this README or add additional documentation.

To contribute:

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request
