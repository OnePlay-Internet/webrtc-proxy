#include "gst.h"

#include <gst/app/gstappsrc.h>

GMainLoop *gstreamer_send_main_loop = NULL;
void gstreamer_send_start_mainloop(void) {
  gstreamer_send_main_loop = g_main_loop_new(NULL, FALSE);
  g_main_loop_run(gstreamer_send_main_loop);
}

static gboolean gstreamer_send_bus_call(GstBus *bus, GstMessage *msg, gpointer data) {
  switch (GST_MESSAGE_TYPE(msg)) {

  case GST_MESSAGE_EOS:
    g_print("End of stream\n");
    exit(1);
    break;

  case GST_MESSAGE_ERROR: {
    gchar *debug;
    GError *error;

    gst_message_parse_error(msg, &error, &debug);
    g_free(debug);

    g_printerr("Error: %s\n", error->message);
    g_error_free(error);
    exit(1);
  }
  default:
    break;
  }

  return TRUE;
}

GstFlowReturn gstreamer_send_new_sample_handler(GstElement *object, gpointer user_data) {
  GstSample *sample = NULL;
  GstBuffer *buffer = NULL;
  gsize copy_size = 0;
  char copy[500] = {0};

  g_signal_emit_by_name (object, "pull-sample", &sample);
  if (sample) {
    buffer = gst_sample_get_buffer(sample);
    if (buffer) {
      copy_size = gst_buffer_get_size(buffer);
      gst_buffer_extract(buffer, 0, (gpointer)copy, copy_size);
      if(copy || copy_size)
        goHandlePipelineBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer));
    }
    gst_sample_unref (sample);
  }
  return GST_FLOW_OK;
}

GstElement *gstreamer_send_create_pipeline(char *pipeline) {
  gst_init(NULL, NULL);
  GError *error = NULL;
  GstElement* el = gst_parse_launch(pipeline, &error);
  return el;
}

void gstreamer_send_start_pipeline(GstElement *pipeline) {
  GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
  gst_bus_add_watch(bus, gstreamer_send_bus_call, NULL);
  gst_object_unref(bus);

  GstElement *appsink = gst_bin_get_by_name(GST_BIN(pipeline), "appsink");
  g_object_set(appsink, "emit-signals", TRUE, NULL);
  g_signal_connect(appsink, "new-sample", G_CALLBACK(gstreamer_send_new_sample_handler), NULL);
  gst_object_unref(appsink);

  gst_element_set_state(pipeline, GST_STATE_PLAYING);
}

void gstreamer_send_stop_pipeline(GstElement *pipeline) {
  gst_element_set_state(pipeline, GST_STATE_NULL);
}