import fastapi
from fastapi.responses import Response, FileResponse
from fastapi.middleware.cors import CORSMiddleware
from fastapi.staticfiles import StaticFiles
import uvicorn
import os

app = fastapi.FastAPI()

origins = [
    "*"
]

app.add_middleware(
    CORSMiddleware,
    allow_origins=origins,
    allow_credentials=True,
    allow_methods=["GET"],
    allow_headers=["*"],
)

# Define a path to the directory containing your static files (CSS, JS, images)
static_folder_path = "./38_North_html_page_files"

# Mount the static files directory to a specific route
app.mount("/38_North_html_page_files", StaticFiles(directory=static_folder_path), name="static")

@app.get("/test/rss_feed")
async def return_demo_xml_page():
    "Returning an XML page of an RSS Feed for testing"
    xml_path: str = "./38_north_test.rss"
    
    if not os.path.exists(xml_path):
        return Response(content="XML File not found", status_code=404)
    
    with open(xml_path, "rb") as f:
        xml_data = f.read()

    response = Response(content=xml_data, media_type="application/xml")
    return response

@app.get("/test/html_page")
async def return_demo_html_page():
    "Returning an html page for testing"

    html_path: str = "./38_North_html_page.html"

    if not os.path.exists(html_path):
        return Response(content="HTML File not found", status_code=404)
    
    return  FileResponse(path=html_path)

if __name__ == '__main__':
    uvicorn.run(app, host='0.0.0.0', port=8000)