# SPK2 Database System

A sophisticated database management system designed for handling JAMB (Joint Admissions and Matriculation Board) candidate data with advanced querying capabilities, including natural language processing features.

## Features

- **Advanced Data Management**
  - Candidate information storage and retrieval
  - Institution management with historical tracking
  - Course and subject tracking
  - Performance metrics analysis

- **Natural Language Querying**
  - AI-powered query processing
  - Intelligent data retrieval
  - User-friendly interface for complex queries

- **Data Analysis Tools**
  - Performance metrics visualization
  - Geographic distribution analysis
  - Gender statistics
  - Course competitiveness analysis
  - Institution rankings
  - Subject correlation studies

- **Data Import/Export**
  - CSV data import functionality
  - Failed import analysis
  - Data validation and verification

## Technology Stack

- **Backend**: Go (Golang)
- **Database**: PostgreSQL
- **Additional Components**: Python for specific processing tasks
- **Dependencies**: See `go.mod` and `requirements.txt`

## Project Structure

```
spk2_db/
├── main.go           # Main application entry point
├── schema.sql        # Database schema definition
├── models/           # Database models and structures
├── nlquery/          # Natural language query processing
├── importer/         # Data import functionality
├── migrations/       # Database migrations
├── csv/             # CSV data files
└── query_tables/    # Query-related table definitions
```

## Setup

1. **Prerequisites**
   - Go 1.x
   - PostgreSQL 16.x
   - Python 3.x (for specific components)

2. **Environment Configuration**
   Create a `.env` file with the following variables:
   ```
   DB_HOST=your_host
   DB_PORT=your_port
   DB_USER=your_user
   DB_PASSWORD=your_password
   DB_NAME=your_database
   ```

3. **Installation**
   ```bash
   # Clone the repository
   git clone https://github.com/nonsonwune/spk2_db.git
   cd spk2_db

   # Install Go dependencies
   go mod download

   # Install Python dependencies
   pip install -r requirements.txt
   ```

4. **Database Setup**
   ```bash
   # Apply database schema
   psql -U your_user -d your_database -f schema.sql
   ```

## Usage

The system provides various functionalities through an interactive menu:

1. **Candidate Management**
   - Search candidates
   - View top performers
   - Analyze performance metrics

2. **Statistical Analysis**
   - Gender distribution
   - Geographic analysis
   - Subject correlations
   - Course competitiveness

3. **Institution Analytics**
   - Institution rankings
   - Faculty performance
   - Regional performance

4. **Data Import**
   - Candidate data import
   - Course data import
   - Failed import analysis

5. **Natural Language Queries**
   - Ask questions in natural language
   - Get intelligent responses based on database content

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is proprietary and confidential. All rights reserved.

## Contact

Nonso Nwune - [GitHub](https://github.com/nonsonwune)

Project Link: [https://github.com/nonsonwune/spk2_db](https://github.com/nonsonwune/spk2_db)
