package com.octara.maps.ui.screens.map

import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import org.maplibre.android.camera.CameraPosition
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.MapView
import org.maplibre.android.maps.Style

private const val OSM_STYLE_JSON = """
{
  "version": 8,
  "sources": {
    "osm-raster-tiles": {
      "type": "raster",
      "tiles": [
        "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
      ],
      "tileSize": 256
    }
  },
  "layers": [
    {
      "id": "osm-tiles",
      "type": "raster",
      "source": "osm-raster-tiles",
      "minzoom": 0,
      "maxzoom": 19
    }
  ]
}
"""

@Composable
fun TransitMap(
    modifier: Modifier = Modifier,
    initialCenter: LatLng = LatLng(48.8566, 2.3522),
    initialZoom: Double = 11.0
) {
    val context = LocalContext.current
    val lifecycle = LocalLifecycleOwner.current.lifecycle

    val mapView = remember {
        MapView(context).apply {
            onCreate(null)
            getMapAsync { map ->
                map.uiSettings.isLogoEnabled = false
                map.uiSettings.isAttributionEnabled = false

                map.setStyle(Style.Builder().fromJson(OSM_STYLE_JSON)) {
                    map.cameraPosition = CameraPosition.Builder()
                        .target(initialCenter)
                        .zoom(initialZoom)
                        .build()
                }
            }
        }
    }

    DisposableEffect(lifecycle, mapView) {
        val observer = LifecycleEventObserver { _, event ->
            when (event) {
                Lifecycle.Event.ON_START -> mapView.onStart()
                Lifecycle.Event.ON_RESUME -> mapView.onResume()
                Lifecycle.Event.ON_PAUSE -> mapView.onPause()
                Lifecycle.Event.ON_STOP -> mapView.onStop()
                Lifecycle.Event.ON_DESTROY -> mapView.onDestroy()
                else -> {}
            }
        }
        lifecycle.addObserver(observer)
        onDispose {
            lifecycle.removeObserver(observer)
        }
    }

    AndroidView(
        factory = { mapView },
        modifier = modifier
    )
}
