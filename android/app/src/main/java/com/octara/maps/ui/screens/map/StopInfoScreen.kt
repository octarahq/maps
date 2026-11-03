package com.octara.maps.ui.screens.map

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.ui.theme.AppDimensions
import com.octara.maps.utils.Api

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StopInfoScreen(
    stopId: String,
    modifier: Modifier = Modifier,
    onBack: (() -> Unit)? = null
) {
    val router = LocalRouter.current
    val scaffoldState = rememberBottomSheetScaffoldState()
    var isLoading by remember { mutableStateOf(true) }
    var stopInfo by remember { mutableStateOf<Api.StopInfoResponse?>(null) }

    LaunchedEffect(stopId) {
        isLoading = true
        stopInfo = com.octara.maps.utils.Api().getStopById(stopId)
        isLoading = false
    }

    BottomSheetScaffold(
        scaffoldState = scaffoldState,
        sheetShape = AppDimensions.BottomSheetShape,
        sheetPeekHeight = AppDimensions.BottomSheetPeekHeight,
        sheetShadowElevation = AppDimensions.BottomSheetElevation,
        sheetContent = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    val info = stopInfo
                    if (info == null && isLoading) {
                        Text(
                            text = "Chargement...",
                            style = MaterialTheme.typography.titleLarge,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    } else if (info == null) {
                        Text(
                            text = "Erreur lors du chargement de l'arrêt",
                            style = MaterialTheme.typography.titleLarge,
                            color = MaterialTheme.colorScheme.error
                        )
                    } else {
                        Text(
                            text = info.toString(),
                            style = MaterialTheme.typography.titleLarge,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    }
                }
            }
        },
        modifier = modifier.fillMaxSize()
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            TransitMap(modifier = Modifier.fillMaxSize())
            IconButton(
                onClick = { onBack?.invoke() ?: router.goBack() },
                modifier = Modifier.align(Alignment.TopStart)
            ) {
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = "Retour",
                    tint = MaterialTheme.colorScheme.onSurface
                )
            }
        }
    }
}
