package com.octara.maps.utils

import android.annotation.SuppressLint
import android.content.Context
import android.content.pm.PackageManager
import android.os.Looper
import androidx.core.content.ContextCompat
import com.google.android.gms.location.LocationCallback
import com.google.android.gms.location.LocationRequest
import com.google.android.gms.location.LocationResult
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlin.coroutines.resume

object LocationManager {
    private val _currentLocation = MutableStateFlow<Pair<Float, Float>?>(null)
    val currentLocation: StateFlow<Pair<Float, Float>?> = _currentLocation.asStateFlow()

    private val _lastFixMillis = MutableStateFlow(0L)
    val lastFixMillis: StateFlow<Long> = _lastFixMillis.asStateFlow()

    private var isTracking = false
    private var highAccuracyMode = false
    private var locationCallback: LocationCallback? = null
    private var sysListener: android.location.LocationListener? = null

    private fun publishLocation(loc: android.location.Location, freshFix: Boolean) {
        _currentLocation.value = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
        val ts = if (freshFix) System.currentTimeMillis() else if (loc.time > 0L) loc.time else System.currentTimeMillis()
        if (ts >= _lastFixMillis.value || freshFix) {
            _lastFixMillis.value = ts
        }
    }

    fun hasLocationPermission(context: Context): Boolean {
        val hasFine = ContextCompat.checkSelfPermission(
            context, android.Manifest.permission.ACCESS_FINE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED
        val hasCoarse = ContextCompat.checkSelfPermission(
            context, android.Manifest.permission.ACCESS_COARSE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED
        return hasFine || hasCoarse
    }

    @SuppressLint("MissingPermission")
    fun setHighAccuracyNavigation(context: Context, enabled: Boolean) {
        if (highAccuracyMode == enabled && isTracking) return
        highAccuracyMode = enabled
        stopTracking(context)
        startLocationUpdates(context)
    }

    @SuppressLint("MissingPermission")
    private fun stopTracking(context: Context) {
        val appCtx = context.applicationContext
        val fusedClient = LocationServices.getFusedLocationProviderClient(appCtx)
        locationCallback?.let { runCatching { fusedClient.removeLocationUpdates(it) } }
        locationCallback = null
        val sysLm = appCtx.getSystemService(Context.LOCATION_SERVICE) as? android.location.LocationManager
        sysListener?.let { l -> runCatching { sysLm?.removeUpdates(l) } }
        sysListener = null
        isTracking = false
    }

    @SuppressLint("MissingPermission")
    fun kickLocationRefresh(context: Context) {
        if (!hasLocationPermission(context)) return
        val appCtx = context.applicationContext
        val fusedClient = LocationServices.getFusedLocationProviderClient(appCtx)
        val cts = CancellationTokenSource()
        fusedClient.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, cts.token)
            .addOnSuccessListener { loc ->
                if (loc != null) publishLocation(loc, freshFix = true)
            }
        if (!isTracking) {
            startLocationUpdates(appCtx)
        }
    }

    @SuppressLint("MissingPermission")
    fun startLocationUpdates(context: Context) {
        if (!hasLocationPermission(context)) return

        val appCtx = context.applicationContext
        val fusedClient = LocationServices.getFusedLocationProviderClient(appCtx)

        fusedClient.lastLocation.addOnSuccessListener { loc ->
            if (loc != null && _currentLocation.value == null) {
                publishLocation(loc, freshFix = false)
            }
        }

        val cts = CancellationTokenSource()
        fusedClient.getCurrentLocation(Priority.PRIORITY_HIGH_ACCURACY, cts.token)
            .addOnSuccessListener { loc ->
                if (loc != null) {
                    publishLocation(loc, freshFix = true)
                }
            }

        if (isTracking) return

        val intervalMs = if (highAccuracyMode) 3_000L else 10_000L
        val minIntervalMs = if (highAccuracyMode) 1_500L else 5_000L
        val minDistM = if (highAccuracyMode) 0f else 10f
        val priority = if (highAccuracyMode) Priority.PRIORITY_HIGH_ACCURACY else Priority.PRIORITY_BALANCED_POWER_ACCURACY

        val request = LocationRequest.Builder(priority, intervalMs)
            .setMinUpdateIntervalMillis(minIntervalMs)
            .setMinUpdateDistanceMeters(minDistM)
            .apply {
                if (highAccuracyMode) {
                    setMaxUpdateDelayMillis(5_000L)
                }
            }
            .build()

        val callback = object : LocationCallback() {
            override fun onLocationResult(result: LocationResult) {
                result.lastLocation?.let { loc ->
                    publishLocation(loc, freshFix = true)
                }
            }
        }
        locationCallback = callback

        try {
            fusedClient.requestLocationUpdates(request, callback, Looper.getMainLooper())
            isTracking = true
        } catch (_: Exception) {
        }

        val sysLm = appCtx.getSystemService(Context.LOCATION_SERVICE) as? android.location.LocationManager
        if (sysLm != null) {
            val listener = object : android.location.LocationListener {
                override fun onLocationChanged(loc: android.location.Location) {
                    publishLocation(loc, freshFix = true)
                }
                @Deprecated("Deprecated in Java")
                override fun onStatusChanged(provider: String?, status: Int, extras: android.os.Bundle?) {}
                override fun onProviderEnabled(provider: String) {}
                override fun onProviderDisabled(provider: String) {}
            }
            sysListener = listener
            for (p in listOf(
                android.location.LocationManager.NETWORK_PROVIDER,
                android.location.LocationManager.GPS_PROVIDER
            )) {
                runCatching {
                    if (sysLm.isProviderEnabled(p)) {
                        sysLm.requestLocationUpdates(p, minIntervalMs, minDistM, listener, Looper.getMainLooper())
                    }
                }
            }
        }
    }

    fun updateLocation(lat: Float, lon: Float) {
        _currentLocation.value = Pair(lat, lon)
        _lastFixMillis.value = System.currentTimeMillis()
    }

    @SuppressLint("MissingPermission")
    suspend fun fetchCurrentLocation(
        context: Context,
        forceRefresh: Boolean = false
    ): Pair<Float, Float>? {
        if (!hasLocationPermission(context)) return null

        val cached = _currentLocation.value
        if (cached != null && !forceRefresh) return cached

        val sysImmediate = getSystemLastKnownLocation(context)
        if (sysImmediate != null) {
            _currentLocation.value = sysImmediate
        }

        startLocationUpdates(context)

        val fusedClient = LocationServices.getFusedLocationProviderClient(context.applicationContext)

        val fresh = withTimeoutOrNull(3_000L) {
            suspendCancellableCoroutine<Pair<Float, Float>?> { continuation ->
                val cts = CancellationTokenSource()
                fusedClient.getCurrentLocation(
                    Priority.PRIORITY_BALANCED_POWER_ACCURACY,
                    cts.token
                ).addOnSuccessListener { loc ->
                    if (loc != null) {
                        val coords = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
                        publishLocation(loc, freshFix = true)
                        continuation.resume(coords)
                    } else {
                        continuation.resume(null)
                    }
                }.addOnFailureListener {
                    continuation.resume(null)
                }
                continuation.invokeOnCancellation { cts.cancel() }
            }
        }
        if (fresh != null) return fresh

        val fusedLast = withTimeoutOrNull(1_500L) {
            suspendCancellableCoroutine<Pair<Float, Float>?> { continuation ->
                fusedClient.lastLocation
                    .addOnSuccessListener { loc ->
                        if (loc != null) {
                            val coords = Pair(loc.latitude.toFloat(), loc.longitude.toFloat())
                            publishLocation(loc, freshFix = false)
                            continuation.resume(coords)
                        } else {
                            continuation.resume(null)
                        }
                    }
                    .addOnFailureListener {
                        continuation.resume(null)
                    }
            }
        }
        if (fusedLast != null) return fusedLast

        return getSystemLastKnownLocation(context)?.also {
            _currentLocation.value = it
        } ?: _currentLocation.value
    }

    @SuppressLint("MissingPermission")
    private fun getSystemLastKnownLocation(context: Context): Pair<Float, Float>? {
        if (!hasLocationPermission(context)) return null
        val sysLm = context.applicationContext.getSystemService(Context.LOCATION_SERVICE) as? android.location.LocationManager
            ?: return null
        val providers = listOf(
            android.location.LocationManager.GPS_PROVIDER,
            android.location.LocationManager.NETWORK_PROVIDER,
            android.location.LocationManager.PASSIVE_PROVIDER
        )
        var best: android.location.Location? = null
        for (provider in providers) {
            val loc = runCatching { sysLm.getLastKnownLocation(provider) }.getOrNull() ?: continue
            if (best == null || loc.time > best.time) {
                best = loc
            }
        }
        return best?.let { Pair(it.latitude.toFloat(), it.longitude.toFloat()) }
    }
}
