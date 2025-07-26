#include <gst/gst.h>
#include <gst/rtsp-server/rtsp-server.h>

#define DEVICE "/dev/video0"
#define WIDTH 1920
#define HEIGHT 1080
#define FRAMERATE "30/1"
#define PORT "8554"
#define MOUNTPOINT "/stream"

int main(int argc, char *argv[]) {
    GMainLoop *loop;
    GstRTSPServer *server;
    GstRTSPMediaFactory *factory;
    
    gst_init(&argc, &argv);
    
    loop = g_main_loop_new(NULL, FALSE);
    server = gst_rtsp_server_new();
    gst_rtsp_server_set_service(server, PORT);
    
    factory = gst_rtsp_media_factory_new();
    
    gchar *launch_str = g_strdup_printf(
        "( v4l2src device=%s ! "
        "image/jpeg,width=%d,height=%d,framerate=%s ! "
        "jpegparse ! vaapijpegdec ! "
        "vaapipostproc ! videoconvert ! queue ! "
        "vaapih264enc ! h264parse ! "
        "rtph264pay name=pay0 config-interval=1 pt=96 )",
        DEVICE, WIDTH, HEIGHT, FRAMERATE);
    
    gst_rtsp_media_factory_set_launch(factory, launch_str);
    gst_rtsp_media_factory_set_shared(factory, TRUE);
    g_free(launch_str);
    
    GstRTSPMountPoints *mounts = gst_rtsp_server_get_mount_points(server);
    gst_rtsp_mount_points_add_factory(mounts, MOUNTPOINT, factory);
    g_object_unref(mounts);
    
    gst_rtsp_server_attach(server, NULL);
    
    g_print("RTSP сервер запущен на rtsp://%s:%s%s\n", 
           "0.0.0.0", PORT, MOUNTPOINT);
    
    g_main_loop_run(loop);
    
    g_main_loop_unref(loop);
    
    return 0;
}